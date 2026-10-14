package onnx

import "math"

// MKLSinAVX512 reproduces Intel VML_HA Sin, worker RVA 0x529fd0,
// using software float32 FMA and the original fixed-point reduction tables.
func MKLSinAVX512(x float32) float32 {
	bits := math.Float32bits(x)
	abs := bits & 0x7fffffff
	if abs >= 0x7f800000 {
		if abs > 0x7f800000 {
			return math.Float32frombits(bits | 0x400000)
		}
		return math.Float32frombits(0xffc00000)
	}
	a := math.Float32frombits(abs)
	if a > 10000 {
		return mklSinLargeAVX512(bits)
	}
	f := math.Float32frombits
	biased := fma32(f(0x3ea2f983), a, 12582912)
	n := float64(float32(biased - 12582912))
	sign := (math.Float32bits(biased) << 31) ^ (bits & 0x80000000)
	r := math.FMA(-3.1415926535846666, n, float64(a))
	r = math.FMA(-5.126688303189038e-12, n, r)
	z := float64(r * r)
	p := math.FMA(2.605780638722852e-06, z, -0.00019809602902257352)
	p = math.FMA(p, z, 0.00833306624608569)
	p = math.FMA(p, z, -0.1666665955042783)
	p = float64(p * z)
	y := float32(math.FMA(p, r, r))
	return f(math.Float32bits(y) ^ sign)
}

func mklSinLargeAVX512(bits uint32) float32 {
	f := math.Float32frombits
	table := mklSinReduction[(bits>>23)&255]
	mantissa := uint64((bits & 0x7fffff) + 0x800000)
	product := uint64(table[2]) * mantissa
	product = uint64(table[1])*mantissa + (product >> 32)
	b := uint32(product)
	a := uint32(uint64(table[0])*mantissa + (product >> 32))
	sign := bits & 0x80000000
	high := f((a >> 9) | (0x3f800000 ^ sign))
	biased := float32(49152 + high)
	high = float32(high - float32(biased-49152))
	mid := f((b >> 18) | ((a & 0x1ff) << 14) | (sign ^ 0x34000000))
	mid = float32(mid - f(sign^0x34000000))
	low := f(((b & 0x3ffff) << 5) | (sign ^ 0x28800000))
	low = float32(low - f(sign^0x28800000))
	total := float32(high + mid)
	mid = float32(mid + float32(high-total))
	low = float32(mid + low)
	r := float32(total * f(0x40c90fdb))
	residual := fma32(f(0x40c90fdb), total, -r)
	residual = fma32(total, f(0xb43bbd2e), residual)
	residual = fma32(low, f(0x40c90fdb), residual)
	z := float32(r * r)
	sp := fma32(f(0x3c08885c), z, f(0xbe2aaaab))
	sp = float32(sp * z)
	t := mklSinTrig[math.Float32bits(biased)&255]
	cos, sine, sineLow, cosBase := f(t[0]), f(t[1]), f(t[2]), f(t[3])
	s0 := fma32(r, cosBase, sine)
	cosSum := float32(cos + cosBase)
	e0 := float32(sine - s0)
	s1 := fma32(r, cos, s0)
	e0 = fma32(cosBase, r, e0)
	e1 := float32(s0 - s1)
	e1 = fma32(r, cos, e1)
	err := float32(e1 + e0)
	sp = float32(sp * r)
	cp := fma32(f(0x3d2aaa7c), z, -0.5)
	cosSum = fma32(-r, sine, cosSum)
	cp = float32(cp * z)
	err = fma32(sp, cosSum, err)
	out := fma32(residual, cosSum, sineLow)
	out = fma32(cp, sine, out)
	out = float32(out + err)
	return float32(s1 + out)
}
