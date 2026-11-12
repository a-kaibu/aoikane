package onnx

import (
	"runtime"
	"sync/atomic"
	"unsafe"
)

// Fast math trades bit-exact reproduction of the original engine for
// speed: kernels may reorder floating-point accumulation. Callers that opt
// in pass OriginalKernelProfile.Reordered; this is the process default.
var fastMath atomic.Bool

// SetFastMath sets the process default for reordered (faster, not
// bit-exact) kernels.
func SetFastMath(on bool) { fastMath.Store(on) }

// FastMath reports the process default for reordered kernels.
func FastMath() bool { return fastMath.Load() }

// runFast is a direct convolution with free accumulation order. It returns
// false when the shape is not handled.
func (c *Conv1D) runFast(input []float32, length int) ([]float32, bool) {
	return c.runFastStrided(input, length, c.In*c.Kernel, 1)
}

// runFastStrided reads the weight of output channel o at reduction step j
// from c.weight[o*wcol+j*wstep].
func (c *Conv1D) runFastStrided(input []float32, length, wcol, wstep int) ([]float32, bool) {
	stride, dilation := max(1, c.Stride), max(1, c.Dilation)
	if (!useAVX2FMA && !wasmSIMDAvailable) || stride != 1 || (c.Out%4 != 0 && !(wasmSIMDAvailable && c.Out == 1)) || c.Left < 0 || c.Right < 0 || len(input) != c.In*length {
		return nil, false
	}
	n := c.Left + length + c.Right - (c.Kernel-1)*dilation
	if n < 24 {
		// Short convolutions (FeatNet's 11-frame encoder) stay on the exact
		// kernels: reordering them moves some output samples by two LSBs.
		return nil, false
	}
	extra := 0
	right := c.Right + extra
	plen := c.Left + length + right
	nt := n + extra
	steps := c.In * c.Kernel
	if (c.Out-1)*wcol+(steps-1)*wstep >= len(c.weight) {
		return nil, false
	}
	src := input
	if c.Left != 0 || right != 0 {
		src = getFloatsUncleared(c.In * plen)
		parallelRows(c.In, plen, func(lo, hi int) {
			for ch := lo; ch < hi; ch++ {
				row := src[ch*plen : (ch+1)*plen]
				clear(row[:c.Left])
				copy(row[c.Left:], input[ch*length:(ch+1)*length])
				clear(row[c.Left+length:])
			}
		})
	}
	xoff := make([]int32, steps)
	for ci := 0; ci < c.In; ci++ {
		for k := 0; k < c.Kernel; k++ {
			xoff[ci*c.Kernel+k] = int32(ci*plen + k*dilation)
		}
	}
	bias := make([]float32, c.Out)
	copy(bias, c.bias)
	y := getFloatsUncleared(c.Out * nt)
	if fastConvBulk(y, nt, src, xoff, c.weight, bias, steps, wcol, wstep) {
		if len(src) != len(input) || unsafe.SliceData(src) != unsafe.SliceData(input) {
			putFloats(src)
		}
		return y, true
	}
	if c.Out%4 != 0 {
		if len(src) != len(input) || unsafe.SliceData(src) != unsafe.SliceData(input) {
			putFloats(src)
		}
		putFloats(y)
		return nil, false
	}
	tiles := (nt + fastConvTile - 1) / fastConvTile
	groups := c.Out / 4
	// Tasks cover a group of output channels over a run of tiles.
	// Split into enough tasks to balance the workers, but keep each task
	// at a few hundred thousand FMAs so scheduling stays negligible.
	workers := runtime.GOMAXPROCS(0)
	per := (tiles*groups + 4*workers - 1) / (4 * workers)
	per = max(per, (1<<18)/(steps*4*fastConvTile))
	per = max(1, min(tiles, per))
	runs := (tiles + per - 1) / per
	parallelFor(groups*runs, func(task int) {
		g, r := task/runs, task%runs
		w := &c.weight[g*4*wcol]
		yb := y[g*4*nt:]
		for t := r * per; t < min(tiles, (r+1)*per); t++ {
			from := min(t*fastConvTile, nt-fastConvTile)
			fastConvKernel(&yb[from], uintptr(nt), &src[from], &xoff[0], w, &bias[g*4], uintptr(steps), uintptr(wcol), uintptr(wstep))
		}
	})
	if len(src) != len(input) || unsafe.SliceData(src) != unsafe.SliceData(input) {
		putFloats(src)
	}
	if extra > 0 {
		short := getFloatsUncleared(c.Out * n)
		for o := 0; o < c.Out; o++ {
			copy(short[o*n:(o+1)*n], y[o*nt:])
		}
		putFloats(y)
		y = short
	}
	return y, true
}

func (c *Conv1D) runFastIf(input []float32, length int, profile []int) ([]float32, bool) {
	if !reorderedKernels(profile) {
		return nil, false
	}
	return c.runFast(input, length)
}

// fastConvTranspose computes the column matrix of a 1-D transposed
// convolution as a 1x1 convolution and scatters it into the output.
func fastConvTranspose(x, w, b []float32, ci, co, kw, frames, length, stride, left int) ([]float32, bool) {
	columns := co * kw
	if (!useAVX2FMA && !wasmSIMDAvailable) || columns%4 != 0 {
		return nil, false
	}
	c := &Conv1D{In: ci, Out: columns, Kernel: 1, weight: w}
	col, ok := c.runFastStrided(x, frames, 1, columns)
	if !ok {
		return nil, false
	}
	y := getFloatsUncleared(co * length)
	parallelRows(co, kw*frames+length, func(lo, hi int) {
		for ch := lo; ch < hi; ch++ {
			row := y[ch*length : (ch+1)*length]
			for t := range row {
				row[t] = b[ch]
			}
			for k := 0; k < kw; k++ {
				src := col[(ch*kw+k)*frames : (ch*kw+k+1)*frames]
				for t, v := range src {
					if pos := t*stride - left + k; pos >= 0 && pos < length {
						row[pos] += v
					}
				}
			}
		}
	})
	putFloats(col)
	return y, true
}
