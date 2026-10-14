package onnx

import "math"

// MKLExp reproduces the original MKL VML_HA SSE exp worker (RVA 0x41ac20)
// including its scalar exceptional worker at RVA 0x41b880.
func MKLExp(x float32) float32 {
	if math.IsNaN(float64(x)) || x < -87.33654022216797 || x > 87.33654022216797 {
		return mklExpExceptional(x)
	}
	biased := float32(float32(x*184.6649627685547) + 12582912)
	bits := math.Float32bits(biased)
	n := float32(biased - 12582912)
	scale := math.Float32frombits(((bits + 0x3f80) << 16) & 0x7f800000)
	table := mklExpTable[bits&127]
	high, low := math.Float32frombits(table[0]), math.Float32frombits(table[1])
	r := float32(x - float32(n*0.0054168701171875))
	r = float32(r - float32(n*-1.6577690757912933e-06))
	p := float32(r + float32(float32(r*r)*0.5))
	p = float32(p * high)
	p = float32(low + p)
	p = float32(high + p)
	return float32(scale * p)
}

func mklExpExceptional(x float32) float32 {
	return mklExpExceptionalReduction(x, false)
}

func mklExpExceptionalReduction(x float32, highFirst bool) float32 {
	bits := math.Float32bits(x)
	if bits&0x7f800000 == 0x7f800000 {
		if bits == 0xff800000 {
			return 0
		}
		if bits&0x7fffff != 0 {
			return math.Float32frombits(bits | 0x400000)
		}
		return x
	}
	if x > 88.72283172607422 {
		return float32(math.Inf(1))
	}
	if x < -103.97207641601562 {
		return 0
	}
	y := float64(x)
	biased := float64(float64(y*92.33248261689366) + 6755399441055744.0)
	n := float64(biased - 6755399441055744.0)
	index := int64(n)
	exponent := index >> 6
	table := mklExpExceptionalTable[index&63]
	high, low := math.Float64frombits(table[0]), math.Float64frombits(table[1])
	// The scalar worker subtracts the low reduction term before the high.
	r := float64(y - float64(n*2.572804622327669e-14))
	r = float64(r - float64(n*0.010830424696223417))
	if highFirst {
		// The Intel scalar worker reverses the two separate subtractions.
		r = float64(y - float64(n*0.010830424696223417))
		r = float64(r - float64(n*2.572804622327669e-14))
	}
	p := float64(float64(0.0013888870459233254*r) + 0.008333341995140497)
	p = float64(float64(p*r) + 0.04166666666677052)
	p = float64(float64(p*r) + 0.1666666666665788)
	p = float64(float64(p*r) + 0.5)
	p = float64(p * r)
	p = float64(p * r)
	p = float64(p + r)
	p = float64(p + low)
	p = float64(p * high)
	if x >= -87.33654022216797 {
		return float32(float64(float64(p+high) * math.Float64frombits(uint64(exponent+1023)<<52)))
	}
	// Scale the split result by 2^60 before compensating the sum, then
	// scale both pieces down. This preserves the original subnormal rounding.
	scale := math.Float64frombits(uint64(exponent+1083) << 52)
	a, b := float64(p*scale), float64(high*scale)
	sum := float64(a + b)
	err := float64(float64(b-sum) + a)
	temp := float64(sum * 6442450944.0)
	rounded := float64(float64(sum+temp) - temp)
	err = float64(float64(sum-rounded) + err)
	return float32(float64(float64(rounded*0x1p-60) + float64(err*0x1p-60)))
}
