package onnx

import "fmt"

func originalProjectionFMAShape(in, out, rows int) bool {
	return rows >= 2 && ((in == 64 && (out == 2 || out == 1)) || (in == 256 && (out == 5 || out == 60)))
}

// RunOriginalProjectionFMA reconstructs Intel FeatNet's FMA projections.
// Single-output projections use the 16-row body of AVX512 SGEMV (856e20),
// with its reordered eight-term updates at 858a10 and serial masked tail.
// Other audited projections reduce serially. Bias is supplied by graph Add.
func (l *Linear) RunOriginalProjectionFMA(input []float32, rows int) ([]float32, error) {
	if !originalProjectionFMAShape(l.In, l.Out, rows) || len(input)%l.In != 0 || len(input)/l.In != rows || len(l.weight) != l.In*l.Out || rows > int(^uint(0)>>1)/l.Out {
		return nil, fmt.Errorf("unsupported original FMA projection shape")
	}
	out := make([]float32, rows*l.Out)
	for row := 0; row < rows; row++ {
		for col := 0; col < l.Out; col++ {
			sum := float32(0)
			if l.In == 64 && l.Out == 1 && row < rows&^15 {
				for k := 0; k < l.In; k += 8 {
					at := row*l.In + k
					// Preserve every native multiplication, FMA and rounding.
					oddHigh := fma32(input[at+5], l.weight[k+5], float32(input[at+7]*l.weight[k+7]))
					evenHigh := fma32(input[at+6], l.weight[k+6], sum)
					evenHigh = fma32(input[at+4], l.weight[k+4], evenHigh)
					evenLow := fma32(input[at], l.weight[k], float32(input[at+2]*l.weight[k+2]))
					oddLow := fma32(input[at+1], l.weight[k+1], float32(input[at+3]*l.weight[k+3]))
					sum = float32(float32(evenHigh+oddHigh) + float32(evenLow+oddLow))
				}
				out[row] = sum
				continue
			}
			for k := 0; k < l.In; k++ {
				sum = fma32(input[row*l.In+k], l.weight[k*l.Out+col], sum)
			}
			out[row*l.Out+col] = sum
		}
	}
	return out, nil
}
