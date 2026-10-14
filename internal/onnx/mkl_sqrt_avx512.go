package onnx

import "math"

// The interpolation and coefficients are adapted from Intel RECIP14.c:
// Copyright (c) 2015, Intel Corporation. See INTEL-APPROXIMATION-LICENSE.
// https://www.intel.com/content/www/us/en/developer/articles/code-sample/reference-implementations-for-ia-approximation-instructions-vrcp14-vrsqrt14-vrcp28-vrsqrt28-vexp2.html

// intelRSqrt14 emulates Intel's RECIP14.c coefficient interpolation. No host
// reciprocal approximation instruction is used. See INTEL-APPROXIMATION-LICENSE.
// x must be positive and finite; the caller handles zero and special values.
func intelRSqrt14(x float32) float32 {
	b := math.Float32bits(x)
	exponent := int((b>>23)&255) - 127
	mantissa := b & 0x7fffff
	if b < 0x800000 {
		exponent = -126
		for mantissa&0x800000 == 0 {
			mantissa <<= 1
			exponent--
		}
		mantissa &= 0x7fffff
	}
	odd := exponent & 1
	normalized := uint32(0x3f800000) + uint32(odd)*0x800000 + mantissa
	var initial float32
	if normalized == 0x3f800000 {
		initial = 1
	} else {
		i := int(mantissa >> 18)
		index := 4*i + 2*odd
		scale := float64(uint64(256) >> odd)
		center := float64(1+odd) + float64(uint64(1)<<odd)*(1.0/64+float64(i)/32)
		value := float64(math.Float32frombits(normalized &^ 255))
		interpolation := float64(intelRSqrt14Coefficients[index+1]) - scale*float64(intelRSqrt14Coefficients[index])*(value-center)
		truncated := math.Float64bits(interpolation) &^ ((uint64(1) << 36) - 1)
		initial = float32(math.Float64frombits(truncated - (uint64(19) << 52)))
	}
	scale := (exponent - odd) / 2
	return math.Float32frombits(uint32(int64(math.Float32bits(initial)) - int64(scale)*(1<<23)))
}

// MKLSqrtAVX512 reproduces the VML HA AVX512 worker at RVA 0x4081c0,
// including its scaled residual branch for extremely small positive values.
// It emulates VRSQRT14PS and FMA in Go, independently of the host CPU.
// The ordinary Intel original's 138,210 captured outputs match this kernel.
func MKLSqrtAVX512(x float32) float32 {
	b := math.Float32bits(x)
	magnitude := b & 0x7fffffff
	if magnitude > 0x7f800000 {
		return math.Float32frombits(b | 0x400000)
	}
	if magnitude == 0 {
		return x
	}
	if b&0x80000000 != 0 {
		return math.Float32frombits(0xffc00000)
	}
	if magnitude == 0x7f800000 {
		return x
	}
	r := intelRSqrt14(x)
	h := float32(r * 0.5)
	s := float32(r * x)
	residual := float32(0)
	// The native residual would lose precision in the subnormal range.
	if r > 0x1p39 {
		scaled := float32(s * 0x1p39)
		residual = fma32(-scaled, scaled, float32(x*0x1p78))
		h = float32(h * 0x1p-78)
	} else {
		residual = fma32(-s, s, x)
	}
	return fma32(residual, h, s)
}

var intelRSqrt14Coefficients = [128]uint32{
	1001, 520261, 707, 367881, 955, 512437, 675, 362349,
	915, 504953, 647, 357056, 877, 497790, 619, 351992,
	841, 490922, 595, 347136, 807, 484331, 571, 342475,
	775, 478001, 549, 337997, 747, 471909, 527, 333693,
	719, 466046, 509, 329545, 693, 460397, 491, 325551,
	669, 454947, 473, 321697, 647, 449688, 457, 317977,
	625, 444606, 441, 314385, 603, 439694, 427, 310910,
	585, 434939, 413, 307549, 567, 430335, 401, 304295,
	549, 425875, 389, 301139, 533, 421551, 377, 298079,
	517, 417355, 365, 295115, 501, 413284, 355, 292237,
	487, 409329, 345, 289439, 473, 405487, 335, 286722,
	461, 401748, 325, 284080, 449, 398111, 317, 281508,
	437, 394571, 309, 279006, 425, 391127, 301, 276569,
	415, 387770, 293, 274195, 403, 384498, 285, 271882,
	393, 381307, 279, 269625, 385, 378194, 271, 267425,
	375, 375155, 265, 265276, 367, 372190, 259, 263178,
}
