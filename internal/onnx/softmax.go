package onnx

import "math"

// MLAS 1.17.1 ComputeSumExp FMA3 coefficients/evaluation order, MIT
// (see MLAS-LICENSE). This handles the non-positive, finite shifted logits
// produced by the audited three-class bou classifier.
func sumExp(x float32) float32 {
	if x < -88.3762626647949 {
		x = -88.3762626647949
	}
	biased := fma32(x, 1.44269504088896341, 12582912)
	m := float32(biased - 12582912)
	x = fma32(m, -6.93145752e-1, x)
	x = fma32(m, -1.42860677e-6, x)
	p := fma32(0x1.694000p-10, x, 0x1.125edcp-7)
	p = fma32(p, x, 0x1.555b5ap-5)
	p = fma32(p, x, 0x1.555450p-3)
	p = fma32(p, x, 0x1.fffff6p-2)
	p = fma32(p, x, 1)
	p = fma32(p, x, 1)
	scale := math.Float32frombits((math.Float32bits(biased) << 23) + 0x3f800000)
	return float32(p * scale)
}

func softmax3(x [3]float32) [3]float32 {
	maximum := x[0]
	for _, v := range x[1:] {
		if v > maximum {
			maximum = v
		}
	}
	var sum float32
	for i := range x {
		x[i] = sumExp(float32(x[i] - maximum))
		sum = float32(sum + x[i])
	}
	scale := float32(1 / sum)
	for i := range x {
		x[i] = float32(x[i] * scale)
	}
	return x
}
