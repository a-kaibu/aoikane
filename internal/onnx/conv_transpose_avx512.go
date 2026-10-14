package onnx

// The audited 128 -> 64, kernel-eight AVX512 ConvTranspose accumulates both
// contributing spatial positions within each 16-channel reduction, then adds
// that partial to saved output. Bias is added after all input-channel blocks.
// The caller validates the original model geometry and tensor extents.
func originalAVX512TransposeDirect(x, weight, bias []float32, ci, co, kernel, frames, length, stride, left int) []float32 {
	y := make([]float32, co*length)
	for oc := 0; oc < co; oc++ {
		for pos := 0; pos < length; pos++ {
			sum := float32(0)
			for start := 0; start < ci; start += 16 {
				partial := float32(0)
				for kw := 0; kw < kernel; kw++ {
					numerator := pos + left - kw
					if numerator < 0 || numerator%stride != 0 {
						continue
					}
					frame := numerator / stride
					if frame >= frames {
						continue
					}
					for ic := start; ic < min(start+16, ci); ic++ {
						partial = fma32(x[ic*frames+frame], weight[(ic*co+oc)*kernel+kw], partial)
					}
				}
				sum = float32(sum + partial)
			}
			y[oc*length+pos] = float32(sum + bias[oc])
		}
	}
	return y
}
