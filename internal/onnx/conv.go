package onnx

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"runtime"
	"sync"
)

// Conv1D is a channel-first, group-one convolution. The MLAS path supports
// unit stride/dilation; original kernels also support the captured vocoder's
// strided and dilated convolutions.
// It covers FeatNet's output channels divisible by four and reproduces the
// bundled MLAS FMA3/AVX kernels with a six-thread numerical partition. The
// computation itself is Go only; numerical order does not depend on host SIMD
// or on the number of Go workers.
type Conv1D struct {
	In, Out, Kernel, Left, Right int
	Stride, Dilation             int // zero retains the original unit defaults
	weight, bias                 []float32
}

// OpenConv1D reads one named Conv and its weights from an ONNX graph. This
// executes only that operation; surrounding graph nodes are not interpreted.
func OpenConv1D(path, name string) (*Conv1D, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	model, err := fields(data)
	if err != nil {
		return nil, err
	}
	gd, err := oneBytes(model, 7)
	if err != nil {
		return nil, err
	}
	graph, err := fields(gd)
	if err != nil {
		return nil, err
	}
	nodes, err := bytesFields(graph, 1)
	if err != nil {
		return nil, err
	}
	var node []field
	for _, nd := range nodes {
		n, err := fields(nd)
		if err != nil {
			return nil, err
		}
		names, err := bytesFields(n, 3)
		if err != nil {
			return nil, err
		}
		if len(names) == 1 && string(names[0]) == name {
			if node != nil {
				return nil, fmt.Errorf("duplicate node name")
			}
			node = n
		}
	}
	if node == nil {
		return nil, fmt.Errorf("Conv node %q not found", name)
	}
	op, err := oneBytes(node, 4)
	if err != nil || string(op) != "Conv" {
		return nil, fmt.Errorf("selected node is not Conv")
	}
	for _, f := range node {
		if f.number == 7 && len(f.data) > 0 && string(f.data) != "ai.onnx" {
			return nil, fmt.Errorf("custom Conv domain unsupported")
		}
	}
	inputs, err := bytesFields(node, 1)
	if err != nil || len(inputs) < 2 || len(inputs) > 3 {
		return nil, fmt.Errorf("invalid Conv inputs")
	}
	initializers, err := bytesFields(graph, 5)
	if err != nil {
		return nil, err
	}
	var w, b []float32
	var dims []int
	for _, td := range initializers {
		t, err := fields(td)
		if err != nil {
			return nil, err
		}
		tn, err := oneBytes(t, 8)
		if err != nil {
			return nil, err
		}
		if string(tn) == string(inputs[1]) {
			w, dims, err = rawFloatTensor(t)
			if err != nil {
				return nil, err
			}
		}
		if len(inputs) == 3 && string(tn) == string(inputs[2]) {
			var bd []int
			b, bd, err = rawFloatTensor(t)
			if err != nil {
				return nil, err
			}
			if len(bd) != 1 {
				return nil, fmt.Errorf("bias must have rank one")
			}
		}
	}
	if len(dims) != 3 || len(w) == 0 {
		return nil, fmt.Errorf("Conv weights must have rank three")
	}
	c := &Conv1D{In: dims[1], Out: dims[0], Kernel: dims[2], weight: w, bias: b}
	if c.Out%4 != 0 {
		return nil, fmt.Errorf("Conv output channels must be a multiple of four")
	}
	if len(inputs) == 3 && len(b) != c.Out {
		return nil, fmt.Errorf("Conv bias shape differs")
	}
	attrs, err := bytesFields(node, 5)
	if err != nil {
		return nil, err
	}
	values := make(map[string]graphAttr)
	for _, ad := range attrs {
		a, err := fields(ad)
		if err != nil {
			return nil, err
		}
		an, err := oneBytes(a, 1)
		if err != nil {
			return nil, err
		}
		name := string(an)
		if _, ok := values[name]; ok {
			return nil, fmt.Errorf("duplicate Conv attribute %s", name)
		}
		v := graphAttr{}
		switch name {
		case "group":
			x, err := intValues(a, 3)
			if err != nil || len(x) != 1 {
				return nil, fmt.Errorf("invalid Conv group")
			}
			v.i = x[0]
		case "strides", "dilations", "kernel_shape", "pads":
			v.ints, err = intValues(a, 8)
			if err != nil {
				return nil, err
			}
		case "auto_pad":
			x, err := oneBytes(a, 4)
			if err != nil {
				return nil, err
			}
			v.text = string(x)
		}
		values[name] = v
	}
	if err := configureConv(c, values); err != nil {
		return nil, err
	}
	return c, nil
}

func intValues(fs []field, number uint64) ([]int64, error) {
	var out []int64
	for _, f := range fs {
		if f.number != number {
			continue
		}
		switch f.wire {
		case 0:
			out = append(out, int64(f.value))
		case 2:
			for p := f.data; len(p) > 0; {
				v, n := binary.Uvarint(p)
				if n <= 0 {
					return nil, fmt.Errorf("invalid packed integers")
				}
				out = append(out, int64(v))
				p = p[n:]
			}
		default:
			return nil, fmt.Errorf("invalid integer wire type")
		}
	}
	return out, nil
}

func rawFloatTensor(fs []field) ([]float32, []int, error) {
	var dtype uint64
	for _, f := range fs {
		if f.number == 2 {
			if f.wire != 0 {
				return nil, nil, fmt.Errorf("invalid FLOAT type field")
			}
			dtype = f.value
		}
		if f.number == 13 || (f.number == 14 && f.value != 0) {
			return nil, nil, fmt.Errorf("external tensor unsupported")
		}
	}
	if dtype != 1 {
		return nil, nil, fmt.Errorf("FLOAT tensor required")
	}
	raw, err := oneBytes(fs, 9)
	if err != nil {
		return nil, nil, err
	}
	d, err := intValues(fs, 1)
	if err != nil {
		return nil, nil, err
	}
	count := len(raw) / 4
	product := 1
	var dims []int
	for _, v := range d {
		if v <= 0 || v > int64(count) || product > count/int(v) {
			return nil, nil, fmt.Errorf("invalid FLOAT dimensions")
		}
		product *= int(v)
		dims = append(dims, int(v))
	}
	if product != count || len(raw)%4 != 0 {
		return nil, nil, fmt.Errorf("FLOAT data length differs")
	}
	values := make([]float32, count)
	for i := range values {
		values[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[4*i:]))
	}
	return values, dims, nil
}

// Run follows MLAS 1.17.1 sgemm.cpp/convolve.cpp and SgemmKernelM1Avx.asm
// (Microsoft, MIT; see MLAS-LICENSE). Numerical partitions remain the same
// across host platforms. Only independent output channels run concurrently.
func (c *Conv1D) Run(input []float32, length int) ([]float32, error) {
	if c.Stride > 1 || c.Dilation > 1 || c.Stride < 0 || c.Dilation < 0 {
		return nil, fmt.Errorf("non-unit Conv stride/dilation requires original kernels")
	}
	if length <= 0 || len(input)%c.In != 0 || len(input)/c.In != length {
		return nil, fmt.Errorf("Conv input shape differs")
	}
	limit := int(^uint(0) >> 1)
	if c.Left > limit-length || c.Right > limit-length-c.Left {
		return nil, fmt.Errorf("Conv padding overflows")
	}
	n := length + c.Left + c.Right - c.Kernel + 1
	if n <= 0 {
		return nil, fmt.Errorf("Conv output length is non-positive")
	}
	if n > limit/c.Out {
		return nil, fmt.Errorf("Conv output is too large")
	}
	kTotal := c.In * c.Kernel
	segmentStride := n
	if c.Kernel > 1 && c.Out <= n {
		threads := convThreads(c.Out, n, kTotal)
		segmentStride = (n + threads - 1) / threads
		if threads > 1 {
			segmentStride = (segmentStride + 15) &^ 15
		}
	}
	out := make([]float32, c.Out*n)
	runChannels := func(first, end int) {
		for o := first; o < end; o++ {
			reduction := 4
			if n == 1 {
				threads := convThreads(c.Out, 1, kTotal)
				start := 0
				for thread := 0; thread < threads; thread++ {
					count := c.Out / threads
					if thread < c.Out%threads {
						count++
					}
					if o >= start && o < start+count {
						local := o - start
						tail := count % 4
						if local >= count-tail {
							reduction = 1
							if tail >= 2 && local < count-tail+2 {
								reduction = 2
							}
						}
						break
					}
					start += count
				}
			}
			for t := 0; t < n; t++ {
				segmentN := min(segmentStride, n-(t/segmentStride)*segmentStride)
				if c.Kernel == 1 && c.Left == 0 && c.Right == 0 && n > c.Out {
					threads := convThreads(c.Out, n, kTotal)
					blocks := (n + 15) / 16
					start := 0
					for thread := 0; thread < threads; thread++ {
						count := blocks / threads
						if thread < blocks%threads {
							count++
						}
						end := min(n, start+count*16)
						if t >= start && t < end {
							segmentN = end - start
							break
						}
						start = end
					}
				}
				strideK, strideN := 128, 128
				if segmentN >= kTotal {
					for strideK/2 >= kTotal {
						strideN *= 2
						strideK /= 2
					}
				} else {
					for strideN > 16 && strideN/2 >= segmentN {
						strideK *= 2
						strideN /= 2
					}
				}
				var dot float32
				var lanes [8]float32
				for start := 0; start < kTotal; start += strideK {
					var partial float32
					for index := start; index < min(kTotal, start+strideK); index++ {
						i, k := index/c.Kernel, index%c.Kernel
						var x float32
						pos := t + k - c.Left
						if pos >= 0 && pos < length {
							x = input[i*length+pos]
						}
						w := c.weight[o*kTotal+index]
						if n == 1 {
							lanes[index%8] = float32(lanes[index%8] + float32(w*x))
						} else {
							partial = fma32(w, x, partial)
						}
					}
					if start == 0 {
						dot = partial
					} else {
						dot = float32(dot + partial)
					}
				}
				if n == 1 {
					switch reduction {
					case 4:
						lo := float32(float32(float32(lanes[0]+lanes[1])+lanes[2]) + lanes[3])
						hi := float32(float32(float32(lanes[4]+lanes[5])+lanes[6]) + lanes[7])
						dot = float32(lo + hi)
					case 2:
						lo := float32(float32(lanes[0]+lanes[2]) + float32(lanes[4]+lanes[6]))
						hi := float32(float32(lanes[1]+lanes[3]) + float32(lanes[5]+lanes[7]))
						dot = float32(lo + hi)
					case 1:
						lo := float32(float32(lanes[0]+lanes[1]) + float32(lanes[2]+lanes[3]))
						hi := float32(float32(lanes[4]+lanes[5]) + float32(lanes[6]+lanes[7]))
						dot = float32(lo + hi)
					}
				}
				if c.bias != nil {
					dot = float32(dot + c.bias[o])
				}
				out[o*n+t] = dot
			}
		}
	}
	// Output channels do not share accumulators. Their scheduling does not
	// change the MLAS numerical partitions or the reduction order above.
	workers := min(6, runtime.GOMAXPROCS(0), c.Out)
	if workers <= 1 || float64(c.Out)*float64(n)*float64(kTotal) < 1<<20 {
		runChannels(0, c.Out)
	} else {
		var wg sync.WaitGroup
		wg.Add(workers)
		for worker := 0; worker < workers; worker++ {
			first, end := worker*c.Out/workers, (worker+1)*c.Out/workers
			go func() {
				defer wg.Done()
				runChannels(first, end)
			}()
		}
		wg.Wait()
	}
	return out, nil
}

func convThreads(m, n, k int) int {
	return 1 + int(min(5, float64(m)*float64(n)*float64(k)/(64*1024)))
}
