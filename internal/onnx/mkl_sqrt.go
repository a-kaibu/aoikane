package onnx

import "math"

// MKLSqrt reproduces the original MKL VML HA float kernel, RVA 0x400480.
// Its initial RSQRTPS approximation is emulated for the original Ryzen host,
// so subsequent non-FMA float32 refinement has the same bits on every CPU.
func MKLSqrt(x float32) float32 {
	bits := math.Float32bits(x)
	magnitude := bits & 0x7fffffff
	if magnitude > 0x7f800000 {
		return math.Float32frombits(bits | 0x00400000)
	}
	if magnitude == 0 {
		return x
	}
	if bits&0x80000000 != 0 {
		return math.Float32frombits(0xffc00000)
	}
	if magnitude == 0x7f800000 {
		return x
	}
	// The original worker routes subnormals and the top 4095 finite inputs
	// through its scalar double-precision path to avoid intermediate overflow.
	if bits < 0x00800000 || bits > 0x7f7ff000 {
		return float32(math.Sqrt(float64(x)))
	}
	exponent := int32(bits >> 23)
	index := int((bits & 0x007fffff) >> 11)
	if exponent&1 == 0 {
		index += 4096
	}
	initial := uint32(0x3e800000) + uint32(originalRSqrt[index])<<11
	scale := (exponent - 127) >> 1
	r := math.Float32frombits(uint32(int64(initial) - int64(scale)*(1<<23)))
	s := float32(x * r)
	h := float32(r * 0.5)
	error := float32(0.5 - float32(s*h))
	s = float32(s + float32(error*s))
	h = float32(h + float32(error*h))
	residual := float32(x - float32(s*s))
	return float32(s + float32(h*residual))
}
