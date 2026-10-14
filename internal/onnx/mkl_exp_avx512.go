package onnx

import "math"

// MKLExpAVX512 reproduces the Intel VML_HA worker at RVA 0x4234a0.
// Directed rounding and the split table are evaluated without host SIMD.
func MKLExpAVX512(x float32) float32 {
	if math.Float32bits(x)&mklExpAVX512abs >= mklExpAVX512limit {
		return mklExpExceptionalReduction(x, true)
	}
	f := math.Float32frombits
	biased := expReductionTowardZero(f(mklExpAVX512scale), x, f(mklExpAVX512bias))
	n := float32(biased - f(mklExpAVX512bias))
	index := math.Float32bits(biased) & 31
	r := fma32(-f(mklExpAVX512hi), n, x)
	r = fma32(-f(mklExpAVX512lo), n, r)
	r = f(math.Float32bits(r) & mklExpAVX512mask)
	square := float32(r * r)
	low := f(mklExpAVX512Table[0][index])
	factor := f(mklExpAVX512Table[1][index])
	scaled := float32(math.Ldexp(float64(factor), int(math.Floor(float64(n)))))
	p := fma32(f(mklExpAVX512cubic), r, f(mklExpAVX512half))
	low = float32(low + r)
	p = fma32(square, p, low)
	return fma32(p, scaled, scaled)
}

// The range reduction has a positive, finite result. Recover the exact
// addition residual before rounding it toward zero to the float32 grid.
func expReductionTowardZero(a, b, c float32) float32 {
	product := float64(a) * float64(b)
	sum := product + float64(c)
	virtualC := sum - product
	residual := (product - (sum - virtualC)) + (float64(c) - virtualC)
	result := float32(sum)
	if float64(result) > sum || (float64(result) == sum && residual < 0) {
		return math.Nextafter32(result, 0)
	}
	return result
}
