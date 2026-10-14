// balance2D adaptation: Copyright 2017-2022 Intel Corporation,
// Apache-2.0; see ONEDNN-LICENSE. Adapted for serial execution in Go.
package onnx

import "fmt"

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
	y := make([]float32, n*c.Out)
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
	y := make([]float32, n*c.Out)
	k := c.In * c.Kernel
	// Reproduce balance2D/balance211 from oneDNN (Apache-2.0). Execute
	// the independent worker regions serially; their numeric order is unchanged.
	balance := func(work, workers, worker int) (int, int) {
		q, r := work/workers, work%workers
		start := worker*q + min(worker, r)
		return start, start + q + min(1, max(0, r-worker))
	}
	groups := plan.threadsOC
	big, small, remainder := threads/groups+1, threads/groups, threads%groups
	for thread := 0; thread < threads; thread++ {
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
				a := make([]float32, rows*channels*c.Kernel)
				for ic := 0; ic < channels; ic++ {
					for offset := 0; offset < c.Kernel; offset++ {
						for row := 0; row < rows; row++ {
							pos := (frame+row)*stride + offset*dilation - c.Left
							if pos >= 0 && pos < length {
								a[(ic*c.Kernel+offset)*rows+row] = input[(channel+ic)*length+pos]
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
						return nil, err
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
				}
			}
		}
	}
	return y, nil
}

func (c *Conv1D) runOriginalDNNL(input []float32, length, n int) ([]float32, error) {
	stride, dilation := max(1, c.Stride), max(1, c.Dilation)
	if (c.In >= 8 && c.In%8 != 0) || (c.Out%8 != 0 && c.Out != 1) {
		return nil, fmt.Errorf("unsupported original oneDNN channel blocking")
	}
	y := make([]float32, n*c.Out)
	for col := 0; col < c.Out; col++ {
		for frame := 0; frame < n; frame++ {
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
				y[col*n+frame] = sum
				continue
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
			y[col*n+frame] = sum
		}
	}
	return y, nil
}
