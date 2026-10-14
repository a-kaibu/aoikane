package onnx

import "fmt"

// runOriginalDNNLAVX512 reproduces the observed oneDNN AVX512 common
// direct-convolution order using scalar Go. Each 16-channel input block
// starts with zero; bias or the previous block is added after its FMAs.
// This is deliberately separate from the default AVX2 dispatch until the
// original CPU-specific primitive selection has been reconstructed.
func (c *Conv1D) runOriginalDNNLAVX512(input []float32, length, n int) ([]float32, error) {
	if c.In < 1 || (c.In >= 16 && c.In%16 != 0) || c.Out < 1 || c.Kernel < 1 || c.Kernel > 8 || length < 1 || n < 1 || len(input) != c.In*length || len(c.weight) != c.Out*c.In*c.Kernel || (len(c.bias) != 0 && len(c.bias) != c.Out) {
		return nil, fmt.Errorf("unsupported original AVX512 direct convolution geometry")
	}
	stride, dilation := max(1, c.Stride), max(1, c.Dilation)
	if c.Left < 0 || c.Right < 0 || c.Stride < 0 || c.Dilation < 0 || n != (length+c.Left+c.Right-(c.Kernel-1)*dilation-1)/stride+1 {
		return nil, fmt.Errorf("invalid original AVX512 direct convolution output extent")
	}
	y := make([]float32, c.Out*n)
	for oc := 0; oc < c.Out; oc++ {
		for frame := 0; frame < n; frame++ {
			sum := float32(0)
			if len(c.bias) != 0 {
				sum = c.bias[oc]
			}
			for block := 0; block < c.In; block += 16 {
				partial := float32(0)
				for kw := 0; kw < c.Kernel; kw++ {
					pos := frame*stride + kw*dilation - c.Left
					if pos < 0 || pos >= length {
						continue
					}
					// Native blocks pad the small-channel input with zeros.
					// Padded lanes cannot contribute to logical outputs.
					for ic := block; ic < min(block+16, c.In); ic++ {
						partial = fma32(input[ic*length+pos], c.weight[(oc*c.In+ic)*c.Kernel+kw], partial)
					}
				}
				sum = float32(partial + sum)
			}
			y[oc*n+frame] = sum
		}
	}
	return y, nil
}
