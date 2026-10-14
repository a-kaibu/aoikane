package onnx

import "math"

// MKLTanh reproduces the original PyTorch 2.0.1 MKL VML high-accuracy
// float kernel (mode 0x140102, worker RVA 0x5bc300). It evaluates a
// piecewise polynomial in float64 with separate multiply/add rounding,
// then rounds once to float32 and restores the original sign bit.
func MKLTanh(x float32) float32 {
	bits := math.Float32bits(x)
	magnitude := bits & 0x7fffffff
	if magnitude > 0x7f800000 {
		return math.Float32frombits(bits | 0x00400000)
	}
	if magnitude == 0x7f800000 {
		return math.Float32frombits(0x3f800000 | (bits & 0x80000000))
	}
	index := int64(bits&0x7fe00000) - 0x3de00000
	index = min(max(index, 0), 0x3400000) >> 21
	c := mklTanhPolynomial[index]
	a := float64(math.Float32frombits(magnitude))
	y := c[7]
	for i := 6; i >= 0; i-- {
		y = float64(y * a)
		y = float64(c[i] + y)
	}
	return math.Float32frombits(math.Float32bits(float32(y)) | (bits & 0x80000000))
}
