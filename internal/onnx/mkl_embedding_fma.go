package onnx

import "fmt"

// RunOriginalEmbeddingFMA reconstructs the Intel one-thread FeatNet embedding
// accumulation order for art, nima and niab. Inputs and weights use the same
// logical row-major layout as Linear.Run. Select it explicitly with the
// FMAEmbeddings profile; remaining projections are still reconstructed separately.
func (l *Linear) RunOriginalEmbeddingFMA(input []float32, rows int) ([]float32, error) {
	if l.Out != 256 || (l.In != 5 && l.In != 10 && l.In != 32) || rows < 2 || len(input)%l.In != 0 || len(input)/l.In != rows || len(l.weight) != l.In*l.Out || rows > int(^uint(0)>>1)/l.Out {
		return nil, fmt.Errorf("unsupported original FMA embedding shape")
	}
	out := make([]float32, rows*l.Out)
	for row := 0; row < rows; row++ {
		for col := 0; col < l.Out; col++ {
			sum := float32(0)
			for k := 0; k < l.In; k++ {
				sum = fma32(input[row*l.In+k], l.weight[k*l.Out+col], sum)
			}
			out[row*l.Out+col] = sum
		}
	}
	return out, nil
}
