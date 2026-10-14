package onnx

import "math"

// MKLLog reproduces original MKL VML_HA SSE Ln (worker RVA 0x43b420).
// Its positive subnormal domain is verified exhaustively against original VML.
func MKLLog(x float32) float32 {
	bits := math.Float32bits(x)
	if bits&0x7f800000 == 0x7f800000 {
		if bits&0x7fffff != 0 {
			return math.Float32frombits(bits | 0x400000)
		}
		if bits&0x80000000 != 0 {
			return math.Float32frombits(0xffc00000)
		}
		return x
	}
	if bits&0x7fffffff == 0 {
		return float32(math.Inf(-1))
	}
	if bits&0x80000000 != 0 {
		return math.Float32frombits(0xffc00000)
	}
	if bits < 0x800000 {
		return float32(math.Log(float64(x)))
	}
	reduced := bits - 0x3f2aaaab
	exponent := float32(int32(reduced) >> 23)
	r := float32(math.Float32frombits((reduced&0x7fffff)+0x3f2aaaab) - 1)
	p := float32(float32(-0.12891685962677002*r) + 0.1398443728685379)
	p = float32(float32(p*r) - 0.12184260785579681)
	p = float32(float32(p*r) + 0.140058696269989)
	p = float32(float32(p*r) - 0.16680487990379333)
	p = float32(float32(p*r) + 0.20010405778884888)
	p = float32(float32(p*r) - 0.24999797344207764)
	p = float32(float32(p*r) + 0.3333321511745453)
	p = float32(float32(p*r) - 0.5)
	p = float32(p * r)
	p = float32(p * r)
	p = float32(r + p)
	p = float32(p + float32(exponent*1.428606765330187e-06))
	return float32(p + float32(exponent*0.693145751953125))
}
