package onnx

import "math"

// intelRCP14 emulates Intel RECIP14.c for positive finite inputs.
// Copyright (c) 2015, Intel Corporation. See INTEL-APPROXIMATION-LICENSE.
func intelRCP14(x float32) float32 {
	b := math.Float32bits(x)
	exponent := int(b>>23) - 127
	mantissa := b & 0x7fffff
	if b < 0x800000 {
		exponent = -126
		for mantissa&0x800000 == 0 {
			mantissa <<= 1
			exponent--
		}
		mantissa &= 0x7fffff
	}
	initial := float32(1)
	if mantissa != 0 {
		i := int(mantissa >> 17)
		v := float64(math.Float32frombits(0x3f800000 | (mantissa &^ 127)))
		center := 1 + 1.0/128 + float64(i)/64
		interpolation := float64(intelRCP14Coefficients[2*i+1]) - 256*float64(intelRCP14Coefficients[2*i])*(v-center)
		truncated := math.Float64bits(interpolation) &^ ((uint64(1) << 36) - 1)
		initial = float32(math.Float64frombits(truncated - (uint64(18) << 52)))
	}
	result := int64(math.Float32bits(initial)) - int64(exponent)*(1<<23)
	if result >= 0x7f800000 {
		return float32(math.Inf(1))
	}
	if result < 0x800000 {
		// RECIP14 truncates the subnormal significand instead of rounding it.
		biased := int(result >> 23)
		return math.Float32frombits((0x800000 | uint32(result)&0x7fffff) >> uint(1-biased))
	}
	return math.Float32frombits(uint32(result))
}

// MKLLogAVX512 reconstructs the original VML HA AVX512 Ln worker.
func MKLLogAVX512(x float32) float32 {
	b := math.Float32bits(x)
	if b&0x7fffffff == 0 {
		return float32(math.Inf(-1))
	}
	if b&0x7fffffff > 0x7f800000 {
		return math.Float32frombits(b | 0x400000)
	}
	if b&0x80000000 != 0 {
		return math.Float32frombits(0xffc00000)
	}
	if b == 0x7f800000 {
		return x
	}
	reciprocal := (math.Float32bits(intelRCP14(x)) + mklLogAVX512bias) & mklLogAVX512mask
	tableRow := 0
	var exponent float32
	var r float32
	if reciprocal < 0x800000 || reciprocal >= 0x7f800000 || b < 0x800000 {
		// VGETMANTPS immediate 0x0a normalizes to [0.5, 1).
		// VRNDSCALEPS immediate 0x58 rounds the reciprocal to multiples of 1/32.
		tableRow = 2
		e := int(b>>23) - 127
		m := b & 0x7fffff
		if b < 0x800000 {
			e = -126
			for m&0x800000 == 0 {
				m <<= 1
				e--
			}
			m &= 0x7fffff
		}
		normalized := math.Float32frombits(0x3f000000 | m)
		approx := intelRCP14(normalized)
		coarse := float32(math.RoundToEven(float64(approx)*32) / 32)
		reciprocal = math.Float32bits(coarse)
		exponent = float32(e - (int(reciprocal>>23) - 127))
		r = fma32(normalized, coarse, -1)
	} else {
		exponent = -float32(int(reciprocal>>23) - 127)
		r = fma32(x, math.Float32frombits(reciprocal), -1)
	}
	c := func(v uint32) float32 { return math.Float32frombits(v) }
	index := (reciprocal >> 18) & 31
	high := fma32(c(mklLogAVX512ln2hi), exponent, c(mklLogAVX512Table[tableRow][index]))
	low := fma32(c(mklLogAVX512ln2lo), exponent, c(mklLogAVX512Table[tableRow+1][index]))
	p := fma32(c(mklLogAVX512cubic), r, c(mklLogAVX512quadratic))
	p = fma32(p, r, c(mklLogAVX512half))
	sum := float32(high + r)
	low = fma32(p, float32(r*r), low)
	residual := float32(r - float32(sum-high))
	return float32(sum + float32(low+residual))
}
