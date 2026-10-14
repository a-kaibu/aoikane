package onnx

import "math"

// MKLTanhAVX512 reproduces the original VML HA AVX512 worker at RVA 0x5c54e0.
// Its centered piecewise polynomial uses float32 FMA and split constant terms.
func MKLTanhAVX512(x float32) float32 {
	bits := math.Float32bits(x)
	magnitude := bits & 0x7fffffff
	if magnitude > 0x7f800000 {
		return math.Float32frombits(bits | 0x400000)
	}
	if magnitude == 0x7f800000 {
		return math.Float32frombits(0x3f800000 | (bits & 0x80000000))
	}
	index := min(max(int64(bits&0x7fe00000)-0x3d400000, 0), 0x3e00000) >> 21
	c := func(row int) float32 { return math.Float32frombits(mklTanhAVX512Table[row][index]) }
	r := float32(math.Float32frombits(magnitude) - c(0))
	p := c(8)
	for row := 7; row >= 4; row-- {
		p = fma32(p, r, c(row))
	}
	p = float32(p * r)
	p = fma32(p, r, c(2))
	p = fma32(r, c(3), p)
	p = float32(p + c(1))
	return math.Float32frombits(math.Float32bits(p) | (bits & 0x80000000))
}
