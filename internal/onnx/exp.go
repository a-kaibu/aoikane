package onnx

import "math"

// Exp reproduces MLAS 1.17.1 ComputeExp FMA3 (Microsoft, MIT;
// see MLAS-LICENSE). Explicit float32 rounding and software FMA keep this
// evaluation order independent of the host instruction set.
func Exp(x float32) float32 {
	x = clamp(x, -103.9720840454, 88.7762626647950)
	biased := fma32(x, 1.44269504088896341, 12582912)
	m := float32(biased - 12582912)
	x = fma32(m, -6.93145752e-1, x)
	x = fma32(m, -1.42860677e-6, x)
	exponent := int32(math.Float32bits(biased) << 23)
	normal := min(max(exponent, int32(-1056964608)), int32(1065353216))
	overflowScale := math.Float32frombits(uint32(exponent-normal) + 0x3f800000)
	normalScale := math.Float32frombits(uint32(normal) + 0x3f800000)
	p := fma32(0x1.694000p-10, x, 0x1.125edcp-7)
	p = fma32(p, x, 0x1.555b5ap-5)
	p = fma32(p, x, 0x1.555450p-3)
	p = fma32(p, x, 0x1.fffff6p-2)
	p = fma32(p, x, 1)
	x = float32(x * overflowScale)
	p = fma32(p, x, overflowScale)
	return float32(p * normalScale)
}
