// balance2D adaptation: Copyright 2017-2022 Intel Corporation,
// Apache-2.0; see ONEDNN-LICENSE. Adapted for serial execution in Go.
package onnx

import (
	"fmt"
	"math"
)

// RunOriginal executes the original engine's six-thread MKL and AVX2 oneDNN
// direct/1x1 and im2col GEMM convolution order using Go only.
// Input/output use logical contiguous channel-first storage.
// Original im2col and output allocations are 64-byte aligned; this alignment
// is represented numerically, independently of actual Go allocations.
func (c *Conv1D) RunOriginal(input []float32, length int) ([]float32, error) {
	return c.runOriginal(input, length, 6)
}

func (c *Conv1D) runOriginal(input []float32, length, threads int, avx512Profile ...bool) ([]float32, error) {
	avx512 := len(avx512Profile) > 0 && avx512Profile[0]
	limit := int(^uint(0) >> 1)
	if c.In < 1 || (c.Out < 6 && c.Out != 1) || c.Kernel < 1 || length < 1 || c.Left < 0 || c.Right < 0 || c.In > limit/length || len(input) != c.In*length || c.In > limit/c.Kernel {
		return nil, fmt.Errorf("unsupported original Conv shape")
	}
	k := c.In * c.Kernel
	if c.Out > limit/k || len(c.weight) != c.Out*k || (len(c.bias) != 0 && len(c.bias) != c.Out) || c.Left > limit-length || c.Right > limit-length-c.Left {
		return nil, fmt.Errorf("invalid original Conv parameters")
	}
	if c.Stride < 0 || c.Dilation < 0 {
		return nil, fmt.Errorf("invalid original Conv stride/dilation")
	}
	stride, dilation := max(1, c.Stride), max(1, c.Dilation)
	if c.Kernel-1 > (limit-1)/dilation {
		return nil, fmt.Errorf("original Conv effective kernel overflows")
	}
	effective := (c.Kernel-1)*dilation + 1
	span := length + c.Left + c.Right - effective
	if span < 0 {
		return nil, fmt.Errorf("original Conv effective kernel exceeds padded input")
	}
	n := span/stride + 1
	if n < 1 || n > limit/k || n > limit/c.Out {
		return nil, fmt.Errorf("invalid original Conv output size")
	}
	// The audited vocoder final Conv has one logical output lane in an
	// eight-lane AVX2 block. Output lanes accumulate independently, so the
	// existing direct kernel order also reproduces its unpadded lane.
	if c.Out == 1 && (c.In != 8 || c.Kernel != 7 || stride != 1 || dilation != 1 || c.Left != 3 || c.Right != 3 || len(input) <= 20480) {
		return nil, fmt.Errorf("unsupported original single-output Conv geometry")
	}
	// PyTorch 2.0.1's batch-one, group-one CPU float selector switches
	// to oneDNN above 20480 input elements, except unstrided/undilated
	// 1x1 filters at one thread (Convolution.cpp v2.0.1, use_mkldnn).
	if len(input) > 20480 && (threads > 1 || c.Kernel != 1 || stride != 1 || dilation != 1) {
		// AVX2 direct convolution unrolls three output positions and rejects
		// larger left padding, including most dilated vocoder convolutions.
		maxKernel := 7
		if avx512 {
			maxKernel = 8
		}
		// AVX512's audited dilated layers retain blocked direct accumulation
		// beyond the AVX2 left-padding limit.
		if c.Kernel > maxKernel || (!avx512 && c.Left > min(3, n)) {
			return c.runOriginalDNNLWide(input, length, n, threads)
		}
		if avx512 && c.Kernel != 1 {
			return c.runOriginalDNNLAVX512(input, length, n)
		}
		return c.runOriginalDNNL(input, length, n)
	}
	a := make([]float32, k*n)
	for channel := 0; channel < c.In; channel++ {
		for offset := 0; offset < c.Kernel; offset++ {
			for frame := 0; frame < n; frame++ {
				pos := frame*stride + offset*dilation - c.Left
				if pos >= 0 && pos < length {
					a[(channel*c.Kernel+offset)*n+frame] = input[channel*length+pos]
				}
			}
		}
	}
	y := getFloats(n * c.Out)
	for col := 0; col < c.Out; col++ {
		if len(c.bias) != 0 {
			for frame := 0; frame < n; frame++ {
				y[col*n+frame] = c.bias[col]
			}
		}
	}
	axis := max(n, c.Out)
	for _, part := range originalMKLPartitions(axis, threads) {
		row, rows, col, columns := 0, n, part[0], part[1]
		// Original BLAS geometry is M=frames, N=channels. On equality
		// the planner chooses N, matching the observed original planner.
		if n > c.Out {
			row, rows, col, columns = part[0], part[1], 0, c.Out
		}
		values, err := MKLSmallKNormalN(a[row:], c.weight[col*k:], y[col*n+row:], rows, columns, k, n, k, n, ((col*n+row)%4)*4)
		if err != nil {
			return nil, err
		}
		for channel := 0; channel < columns; channel++ {
			copy(y[(col+channel)*n+row:], values[channel*rows:(channel+1)*rows])
		}
	}
	return y, nil
}

func (c *Conv1D) runOriginalDNNLWide(input []float32, length, n int, profile ...int) ([]float32, error) {
	threads := originalThreadCount(profile)
	stride, dilation := max(1, c.Stride), max(1, c.Dilation)
	plan := originalDNNLStridedPlan(c.In, c.Out, c.Kernel, n, stride, threads)
	y := getFloats(n * c.Out)
	k := c.In * c.Kernel
	// Reproduce balance2D/balance211 from oneDNN (Apache-2.0). The
	// independent worker regions keep their numeric order when run concurrently.
	balance := func(work, workers, worker int) (int, int) {
		q, r := work/workers, work%workers
		start := worker*q + min(worker, r)
		return start, start + q + min(1, max(0, r-worker))
	}
	groups := plan.threadsOC
	big, small, remainder := threads/groups+1, threads/groups, threads%groups
	// Worker regions write disjoint outputs, so they run concurrently.
	errs := make([]error, threads)
	parallelFor(threads, func(thread int) {
		group, worker, workers := 0, 0, small
		if thread < remainder*big {
			group, worker, workers = thread/big, thread%big, big
		} else {
			distance := thread - remainder*big
			group, worker = remainder+distance/small, distance%small
		}
		colStart, colEnd := balance(c.Out, groups, group)
		frameStart, frameEnd := balance(n, workers, worker)
		for frame := frameStart; frame < frameEnd; frame += plan.sp {
			rows := min(plan.sp, frameEnd-frame)
			for channel := 0; channel < c.In; channel += plan.ic {
				channels := min(plan.ic, c.In-channel)
				a := getFloatsUncleared(rows * channels * c.Kernel)
				for ic := 0; ic < channels; ic++ {
					src := input[(channel+ic)*length : (channel+ic+1)*length]
					for offset := 0; offset < c.Kernel; offset++ {
						dst := a[(ic*c.Kernel+offset)*rows : (ic*c.Kernel+offset+1)*rows]
						shift := frame*stride + offset*dilation - c.Left
						if stride == 1 {
							// Rows whose position lies inside the input; the rest stay zero.
							lo, hi := max(0, -shift), min(rows, length-shift)
							if lo < hi {
								clear(dst[:lo])
								copy(dst[lo:hi], src[shift+lo:shift+hi])
								clear(dst[hi:])
							} else {
								clear(dst)
							}
							continue
						}
						clear(dst)
						for row := range dst {
							if pos := row*stride + shift; pos >= 0 && pos < length {
								dst[row] = src[pos]
							}
						}
					}
				}
				beta := float32(0)
				if channel != 0 {
					beta = 1
				}
				for col := colStart; col < colEnd; col += plan.oc {
					columns := min(plan.oc, colEnd-col)
					var values []float32
					var err error
					if rows == 1 && threads == 1 {
						values, err = dnnlGEMVRow(a, c.weight[col*k+channel*c.Kernel:], y[col*n+frame:], columns, channels*c.Kernel, rows, k, n, beta)
					} else {
						values, err = DNNLGEMMNormalN(a, c.weight[col*k+channel*c.Kernel:], y[col*n+frame:], rows, columns, channels*c.Kernel, rows, k, n, beta)
					}
					if err != nil {
						errs[thread] = err
						return
					}
					for oc := 0; oc < columns; oc++ {
						for row := 0; row < rows; row++ {
							value := values[oc*rows+row]
							if channel+channels == c.In && len(c.bias) != 0 {
								value = float32(value + c.bias[col+oc])
							}
							y[(col+oc)*n+frame+row] = value
						}
					}
					putFloats(values)
				}
				putFloats(a)
			}
		}
	})
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return y, nil
}

func (c *Conv1D) runOriginalDNNL(input []float32, length, n int) ([]float32, error) {
	stride, dilation := max(1, c.Stride), max(1, c.Dilation)
	if (c.In >= 8 && c.In%8 != 0) || (c.Out%8 != 0 && c.Out != 1) {
		return nil, fmt.Errorf("unsupported original oneDNN channel blocking")
	}
	y := getFloatsUncleared(n * c.Out) // every output is written below
	// scalar computes one output in the original order. Positions outside
	// the input are skipped rather than multiplied by zero.
	scalar := func(col, frame int) float32 {
		sum := float32(0)
		if len(c.bias) != 0 {
			sum = c.bias[col]
		}
		if c.Kernel == 1 {
			// The blocked AVX2 1x1 kernel reduces 128 channels at a
			// time. Later blocks start from zero, then add saved C.
			pos := frame*stride - c.Left
			if pos >= 0 && pos < length {
				for start := 0; start < c.In; start += 128 {
					partial := float32(0)
					if start == 0 {
						partial = sum
					}
					for channel := start; channel < min(start+128, c.In); channel++ {
						partial = fma32(input[channel*length+pos], c.weight[col*c.In+channel], partial)
					}
					if start == 0 {
						sum = partial
					} else {
						sum = partial + sum
					}
				}
			}
			return sum
		}
		for block := 0; block < c.In; block += 8 {
			for offset := 0; offset < c.Kernel; offset++ {
				pos := frame*stride + offset*dilation - c.Left
				if pos < 0 || pos >= length {
					continue
				}
				for channel := block; channel < min(block+8, c.In); channel++ {
					sum = fma32(input[channel*length+pos], c.weight[(col*c.In+channel)*c.Kernel+offset], sum)
				}
			}
		}
		return sum
	}
	// Interior frames read every kernel position, so each output is the
	// same fixed FMA sequence; consecutive frames run as independent lanes.
	if c.Kernel > 1 && n >= 8 && useChainedKernels && zeroPaddingExact(c.weight) {
		c.runOriginalDNNLPadded(input, length, n, y)
		// Zero outputs may differ in sign; recompute them in the original order.
		parallelRows(c.Out, n, func(lo, hi int) {
			for i := lo * n; i < hi*n; i++ {
				if y[i] == 0 {
					y[i] = scalar(i/n, i%n)
				}
			}
		})
		return y, nil
	}
	// Strided inputs are split into stride phases, so that consecutive
	// output frames read consecutive elements of one phase.
	src, phaseLength := input, length
	if stride > 1 {
		phaseLength = (length + stride - 1) / stride
		src = make([]float32, stride*c.In*phaseLength)
		for channel := 0; channel < c.In; channel++ {
			for pos := 0; pos < length; pos++ {
				src[((pos%stride)*c.In+channel)*phaseLength+pos/stride] = input[channel*length+pos]
			}
		}
	}
	// locate finds input[channel][frame*stride+rel] at src[offset+frame].
	locate := func(channel, rel int) int32 {
		q := rel / stride
		if rel%stride < 0 {
			q--
		}
		return int32(((rel-q*stride)*c.In+channel)*phaseLength + q)
	}
	first := (c.Left + stride - 1) / stride
	end := 0
	if last := length - 1 - (c.Kernel-1)*dilation + c.Left; last >= 0 {
		end = min(n, last/stride+1)
	}
	vector := end - first
	if vector < 8 {
		first, end, vector = 0, 0, 0
	}
	// Each FMA step is one (channel, offset) pair. For 1x1 filters the
	// steps are split into the 128-channel blocks of the original kernel.
	type segment struct {
		xoff []int32
		w0   int // weight index of the first step for column zero
	}
	var segments []segment
	var gathered [][]float32 // per column, weights in step order
	if vector > 0 {
		if c.Kernel == 1 {
			for start := 0; start < c.In; start += 128 {
				var xoff []int32
				for channel := start; channel < min(start+128, c.In); channel++ {
					xoff = append(xoff, locate(channel, -c.Left))
				}
				segments = append(segments, segment{xoff, start})
			}
			gathered = make([][]float32, c.Out)
			for col := range gathered {
				gathered[col] = c.weight[col*c.In : (col+1)*c.In]
			}
		} else {
			var xoff []int32
			var order []int
			for block := 0; block < c.In; block += 8 {
				for offset := 0; offset < c.Kernel; offset++ {
					for channel := block; channel < min(block+8, c.In); channel++ {
						xoff = append(xoff, locate(channel, offset*dilation-c.Left))
						order = append(order, channel*c.Kernel+offset)
					}
				}
			}
			segments = []segment{{xoff, 0}}
			gathered = cachedPacking(c.weight, c.In, c.Out, c.Kernel, 0, func() [][]float32 {
				gathered := make([][]float32, c.Out)
				k := c.In * c.Kernel
				for col := range gathered {
					g := make([]float32, len(order))
					for j, index := range order {
						g[j] = c.weight[col*k+index]
					}
					gathered[col] = g
				}
				return gathered
			})
		}
	}
	// Border frames skip some kernel positions. With 1x1 filters the
	// skipped frames are constant; otherwise each border frame runs its
	// own step sequence with output channels as the vector lanes.
	var steps []int32 // per step: channel*Kernel+offset in step order
	var transposed []float32
	if vector > 0 && c.Kernel > 1 && c.Out%8 == 0 && useChainedKernels && first+(n-end) > 0 {
		for block := 0; block < c.In; block += 8 {
			for offset := 0; offset < c.Kernel; offset++ {
				for channel := block; channel < min(block+8, c.In); channel++ {
					steps = append(steps, int32(channel*c.Kernel+offset))
				}
			}
		}
		transposed = cachedPacking(c.weight, c.In, c.Out, c.Kernel, 1, func() []float32 {
			transposed := make([]float32, len(steps)*c.Out)
			for col := 0; col < c.Out; col++ {
				for j, v := range gathered[col] {
					transposed[j*c.Out+col] = v
				}
			}
			return transposed
		})
	}
	var borders []int
	for frame := 0; frame < first; frame++ {
		borders = append(borders, frame)
	}
	for frame := end; frame < n; frame++ {
		borders = append(borders, frame)
	}
	parallelFor(len(borders), func(i int) {
		frame := borders[i]
		if transposed == nil {
			for col := 0; col < c.Out; col++ {
				y[col*n+frame] = scalar(col, frame)
			}
			return
		}
		var xoff []int32
		var values []float32
		for j, step := range steps {
			channel, offset := int(step)/c.Kernel, int(step)%c.Kernel
			pos := frame*stride + offset*dilation - c.Left
			if pos >= 0 && pos < length {
				xoff = append(xoff, int32(j*c.Out))
				values = append(values, input[channel*length+pos])
			}
		}
		acc := make([]float32, c.Out)
		if len(c.bias) != 0 {
			copy(acc, c.bias)
		}
		fmaChainCol(acc, transposed, 0, xoff, values, 1)
		for col := 0; col < c.Out; col++ {
			y[col*n+frame] = acc[col]
		}
	})
	if vector == 0 {
		return y, nil
	}
	// Interior tiles are multiples of eight frames; a final partial tile
	// is recomputed as the last eight frames, keeping only the new ones.
	type span struct{ from, rows, keep int }
	var spans []span
	const tile = 256
	covered := first
	for covered+8 <= end {
		rows := min(tile, (end-covered)&^7)
		spans = append(spans, span{covered, rows, 0})
		covered += rows
	}
	if covered < end {
		spans = append(spans, span{end - 8, 8, covered - (end - 8)})
	}
	groups := (c.Out + chainCols - 1) / chainCols
	parallelFor(groups, func(group int) {
		colStart := group * chainCols
		cols := min(chainCols, c.Out-colStart)
		acc := make([]float32, chainCols*tile)
		partial := make([]float32, chainCols*tile)
		for _, sp := range spans {
			from, rows := sp.from, sp.rows
			for cc := 0; cc < chainCols; cc++ {
				sum := float32(0)
				if len(c.bias) != 0 {
					sum = c.bias[colStart+min(cc, cols-1)]
				}
				for r := 0; r < rows; r++ {
					acc[cc*rows+r] = sum
				}
			}
			for index, seg := range segments {
				var w [chainCols][]float32
				for cc := range w {
					w[cc] = gathered[colStart+min(cc, cols-1)][seg.w0:]
				}
				if index == 0 {
					fmaChains(acc[:chainCols*rows], rows, src, from, seg.xoff, w, 1, true)
					continue
				}
				clear(partial[:chainCols*rows])
				fmaChains(partial[:chainCols*rows], rows, src, from, seg.xoff, w, 1, true)
				for i := 0; i < chainCols*rows; i++ {
					acc[i] = partial[i] + acc[i]
				}
			}
			for cc := 0; cc < cols; cc++ {
				copy(y[(colStart+cc)*n+from+sp.keep:], acc[cc*rows+sp.keep:(cc+1)*rows])
			}
		}
	})
	return y, nil
}

// zeroPaddingExact reports whether the weights are finite. Then a skipped
// position's fma(0, w, acc) adds a signed zero product, and acc + (±0) == acc
// unless acc is -0 and the product +0. An accumulator that differs this way
// is zero and stays equal in magnitude, so a padded result can only differ
// from the skipping order in the sign of a zero output; such outputs are
// recomputed in the original order.
func zeroPaddingExact(weight []float32) bool {
	return cachedPacking(weight, 0, len(weight), 0, 2, func() bool {
		for _, w := range weight {
			if math.IsInf(float64(w), 0) || w != w {
				return false
			}
		}
		return true
	})
}

// runOriginalDNNLPadded evaluates the direct convolution order on a
// zero-padded copy of the input, so every frame runs the full step sequence
// as vector lanes. See zeroPaddingExact for the zero outputs.
func (c *Conv1D) runOriginalDNNLPadded(input []float32, length, n int, y []float32) {
	stride, dilation := max(1, c.Stride), max(1, c.Dilation)
	span := (n-1)*stride + (c.Kernel-1)*dilation + 1 // padded positions read
	phaseLength := (span + stride - 1) / stride
	src := make([]float32, stride*c.In*phaseLength)
	for channel := 0; channel < c.In; channel++ {
		for pos := 0; pos < length; pos++ {
			p := pos + c.Left
			if p < span {
				src[((p%stride)*c.In+channel)*phaseLength+p/stride] = input[channel*length+pos]
			}
		}
	}
	var xoff, woff []int32
	for block := 0; block < c.In; block += 8 {
		for offset := 0; offset < c.Kernel; offset++ {
			for channel := block; channel < min(block+8, c.In); channel++ {
				rel := offset * dilation
				xoff = append(xoff, int32(((rel%stride)*c.In+channel)*phaseLength+rel/stride))
				woff = append(woff, int32(channel*c.Kernel+offset))
			}
		}
	}
	type span8 struct{ from, rows, keep int }
	var spans []span8
	const tile = 256
	covered := 0
	for covered+8 <= n {
		rows := min(tile, (n-covered)&^7)
		spans = append(spans, span8{covered, rows, 0})
		covered += rows
	}
	if covered < n {
		spans = append(spans, span8{n - 8, 8, covered - (n - 8)})
	}
	k := c.In * c.Kernel
	groups := (c.Out + chainCols - 1) / chainCols
	parallelFor(groups*len(spans), func(task int) {
		group, sp := task/len(spans), spans[task%len(spans)]
		colStart := group * chainCols
		cols := min(chainCols, c.Out-colStart)
		var w [chainCols][]float32
		acc := make([]float32, chainCols*sp.rows)
		for cc := range w {
			col := colStart + min(cc, cols-1)
			w[cc] = c.weight[col*k : (col+1)*k]
			if len(c.bias) != 0 {
				for r := 0; r < sp.rows; r++ {
					acc[cc*sp.rows+r] = c.bias[col]
				}
			}
		}
		fmaChainsIndexed(acc, sp.rows, src, sp.from, xoff, w, woff)
		for cc := 0; cc < cols; cc++ {
			copy(y[(colStart+cc)*n+sp.from+sp.keep:], acc[cc*sp.rows+sp.keep:(cc+1)*sp.rows])
		}
	})
}
