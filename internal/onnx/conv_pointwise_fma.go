package onnx

import "fmt"

// RunOriginalPointwiseFMA reproduces the observed Intel one-thread unit 1x1
// Conv order for up to 256 input channels. Bias is added after the fused dot
// product. Inputs and outputs are channel-first; no native library is used.
func (c *Conv1D) RunOriginalPointwiseFMA(input []float32, length int) ([]float32, error) {
	limit := int(^uint(0) >> 1)
	if c.In < 1 || c.In > 256 || c.Out < 6 || length < 1 || c.Kernel != 1 || c.Left != 0 || c.Right != 0 || c.Stride < 0 || c.Stride > 1 || c.Dilation < 0 || c.Dilation > 1 || c.In > limit/length || len(input) != c.In*length || c.Out > limit/c.In || len(c.weight) != c.In*c.Out || (len(c.bias) != 0 && len(c.bias) != c.Out) || c.Out > limit/length {
		return nil, fmt.Errorf("unsupported original FMA pointwise Conv geometry")
	}
	out := make([]float32, c.Out*length)
	for col := 0; col < c.Out; col++ {
		for frame := 0; frame < length; frame++ {
			sum := float32(0)
			for channel := 0; channel < c.In; channel++ {
				sum = fma32(input[channel*length+frame], c.weight[col*c.In+channel], sum)
			}
			if len(c.bias) != 0 {
				sum = float32(sum + c.bias[col])
			}
			out[col*length+frame] = sum
		}
	}
	return out, nil
}
