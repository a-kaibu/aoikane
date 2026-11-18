package onnx

import (
	"math"
	"math/bits"
)

// float16 expands binary16 storage to binary32 without rounding.
func float16(h uint16) float32 {
	sign := uint32(h&0x8000) << 16
	exp := uint32(h>>10) & 31
	mant := uint32(h & 1023)
	switch exp {
	case 0:
		if mant == 0 {
			return math.Float32frombits(sign)
		}
		shift := bits.LeadingZeros32(mant) - 21
		return math.Float32frombits(sign | uint32(113-shift)<<23 | (mant<<shift&1023)<<13)
	case 31:
		return math.Float32frombits(sign | 255<<23 | mant<<13)
	default:
		return math.Float32frombits(sign | (exp+112)<<23 | mant<<13)
	}
}
