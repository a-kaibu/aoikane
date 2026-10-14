package onnx

import "math"

// MKLSin reproduces original VML_HA SSE Sin, worker RVA 0x503930.
// The main and large-argument paths round each original operation separately.
func MKLSin(x float32) float32 {
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
		return mklSinLarge(bits)
	}
	biased := float32(float32(a*0.31830987334251404) + 12582912)
	n := float64(float32(biased - 12582912))
	sign := (math.Float32bits(biased) << 31) ^ (bits & 0x80000000)
	r := float64(float64(a) - float64(n*3.1415926535846666))
	r = float64(r - float64(n*5.126688303189038e-12))
	z := float64(r * r)
	p := float64(float64(2.605780638722852e-06*z) - 0.00019809602902257352)
	p = float64(float64(p*z) + 0.00833306624608569)
	p = float64(float64(p*z) - 0.1666665955042783)
	p = float64(float64(z*p) * r)
	y := float32(float64(r + p))
	return math.Float32frombits(math.Float32bits(y) ^ sign)
}

func mklSinLarge(bits uint32) float32 {
	table := mklSinReduction[(bits>>23)&255]
	mantissa := uint64((bits & 0x7fffff) + 0x800000)
	// Retain the low 96 bits of the original fixed-point product.
	product := uint64(table[2]) * mantissa
	product = uint64(table[1])*mantissa + (product >> 32)
	b := uint32(product)
	a := uint32(uint64(table[0])*mantissa + (product >> 32))
	sign := bits & 0x80000000
	high := math.Float32frombits((a >> 9) | (0x3f800000 ^ sign))
	biased := float32(49152 + high)
	rounded := float32(biased - 49152)
	high = float32(high - rounded)
	mid := math.Float32frombits((b >> 18) | ((a & 0x1ff) << 14) | (sign ^ 0x34000000))
	mid = float32(mid - math.Float32frombits(sign^0x34000000))
	low := math.Float32frombits(((b & 0x3ffff) << 5) | (sign ^ 0x28800000))
	low = float32(low - math.Float32frombits(sign^0x28800000))
	total := float32(high + mid)
	mid = float32(mid + float32(high-total))
	low = float32(low + mid)
	trimHigh := math.Float32frombits(math.Float32bits(total) & 0xfffff000)
	trimLow := float32(total - trimHigh)
	rHigh := float32(6.283203125 * trimHigh)
	rLow := float32(float32(6.283203125*trimLow) + float32(trimHigh*-1.781781975296326e-05))
	low = float32(float32(low*6.2831854820251465) + float32(trimLow*-1.781781975296326e-05))
	rLow = float32(rLow + low)
	r := float32(rLow + rHigh)
	rLow = float32(rLow + float32(rHigh-r))
	t := mklSinTrig[math.Float32bits(biased)&255]
	cos, sine, sineLow, cosBase := math.Float32frombits(t[0]), math.Float32frombits(t[1]), math.Float32frombits(t[2]), math.Float32frombits(t[3])
	p := float32(r * cosBase)
	s0 := float32(sine + p)
	e0 := float32(sine - s0)
	p1 := float32(r * cos)
	s1 := float32(s0 + p1)
	e1 := float32(s0 - s1)
	e1 = float32(p1 + e1)
	e0 = float32(p + e0)
	z := float32(r * r)
	sp := float32(0.008333291858434677 * z)
	sp = float32(-0.1666666716337204 + sp)
	sp = float32(sp * z)
	sp = float32(sp * r)
	cosSum := float32(cos + cosBase)
	e0 = float32(e0 + e1)
	cp := float32(0.04166649281978607 * z)
	cp = float32(-0.5 + cp)
	cp = float32(z * cp)
	cosSum = float32(cosSum - float32(r*sine))
	out := float32(sineLow + float32(rLow*cosSum))
	out = float32(out + float32(sine*cp))
	e0 = float32(e0 + float32(sp*cosSum))
	out = float32(out + e0)
	return float32(s1 + out)
}
