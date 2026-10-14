package onnx

import "fmt"

// RunOriginalSmallConvFMA reconstructs serial Intel im2col accumulation.
// It covers unit stride/dilation and at most 320 reduction terms. Bias is
// added after the complete channel/kernel dot product, including zero padding.
func (c *Conv1D) RunOriginalSmallConvFMA(input []float32, length int) ([]float32, error) {
	limit := int(^uint(0) >> 1)
	if c.In < 1 || c.Kernel < 1 || c.In > 320/c.Kernel || c.Out < 6 || length < 1 || c.Left < 0 || c.Right < 0 || c.Stride < 0 || c.Stride > 1 || c.Dilation < 0 || c.Dilation > 1 || c.In > limit/length || len(input) != c.In*length || c.Left > limit-length || c.Right > limit-length-c.Left {
		return nil, fmt.Errorf("unsupported original small FMA Conv geometry")
	}
	n := length + c.Left + c.Right - c.Kernel + 1
	k := c.In * c.Kernel
	if n < 1 || c.Out > limit/k || len(c.weight) != c.Out*k || (len(c.bias) != 0 && len(c.bias) != c.Out) || c.Out > limit/n {
		return nil, fmt.Errorf("invalid original small FMA Conv parameters")
	}
	out := make([]float32, c.Out*n)
	for col := 0; col < c.Out; col++ {
		for frame := 0; frame < n; frame++ {
			sum := float32(0)
			for channel := 0; channel < c.In; channel++ {
				for offset := 0; offset < c.Kernel; offset++ {
					pos := frame + offset - c.Left
					x := float32(0)
					if pos >= 0 && pos < length {
						x = input[channel*length+pos]
					}
					sum = fma32(x, c.weight[col*k+channel*c.Kernel+offset], sum)
				}
			}
			if len(c.bias) != 0 {
				sum = float32(sum + c.bias[col])
			}
			out[col*n+frame] = sum
		}
	}
	return out, nil
}
