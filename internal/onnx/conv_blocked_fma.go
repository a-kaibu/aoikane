package onnx

import "fmt"

// RunOriginalBlockedConvFMA reconstructs the audited Intel im2col FMA order.
// Reduction blocks run from the beginning, with a maximum width of 384;
// the final two blocks are balanced. Each partial dot product is added to
// the output initialized with bias. This differs from one serial dot product.
func (c *Conv1D) RunOriginalBlockedConvFMA(input []float32, length int) ([]float32, error) {
	limit := int(^uint(0) >> 1)
	if c.In < 1 || c.Kernel < 1 || !originalBlockedConvShape(c.In, c.Out, c.Kernel) || c.Out < 6 || length < 1 || c.Left < 0 || c.Right < 0 || c.Stride < 0 || c.Stride > 1 || c.Dilation < 0 || c.Dilation > 1 || c.In > limit/length || len(input) != c.In*length || c.Left > limit-length || c.Right > limit-length-c.Left {
		return nil, fmt.Errorf("unsupported original blocked FMA Conv geometry")
	}
	n := length + c.Left + c.Right - c.Kernel + 1
	k := c.In * c.Kernel
	if n < 1 || c.Out > limit/k || len(c.weight) != c.Out*k || (len(c.bias) != 0 && len(c.bias) != c.Out) || c.Out > limit/n {
		return nil, fmt.Errorf("invalid original blocked FMA Conv parameters")
	}
	out := make([]float32, c.Out*n)
	for col := 0; col < c.Out; col++ {
		for frame := 0; frame < n; frame++ {
			sum := float32(0)
			if len(c.bias) != 0 {
				sum = c.bias[col]
			}
			for start := 0; start < k; {
				width := 384
				remaining := k - start
				if remaining <= width {
					width = remaining
				} else if remaining < 2*width {
					width = (remaining + 1) / 2
				}
				partial := float32(0)
				for inner := start; inner < start+width; inner++ {
					channel, offset := inner/c.Kernel, inner%c.Kernel
					pos := frame + offset - c.Left
					x := float32(0)
					if pos >= 0 && pos < length {
						x = input[channel*length+pos]
					}
					partial = fma32(x, c.weight[col*k+inner], partial)
				}
				sum = float32(sum + partial)
				start += width
			}

			out[col*n+frame] = sum
		}
	}
	return out, nil
}

// Keep selection restricted to geometries captured from the ordinary original.
func originalBlockedConvShape(in, out, kernel int) bool {
	switch {
	case in == 64 && out == 128:
		return kernel == 9 || kernel == 17 || kernel == 33
	case in == 64 && out == 16:
		return kernel == 11 || kernel == 15 || kernel == 21 || kernel == 31 || kernel == 41
	case in == 256 && out == 512:
		return kernel == 5
	case in == 512 && out == 256:
		return kernel == 1
	}
	return false
}
