package dsp

import "math"

// originalMagnitudePow reconstructs the ordinary Windows UCRT powf FMA path
// for the model's fixed base 10. The model's clamped power gives exponents
// in [-5, 0]. The native range reduction and cubic order intentionally differ
// from float32(math.Pow(10, float64(exponent))). No native runtime is used.
func originalMagnitudePow(exponent float32) float32 {
	z := float64(exponent) * math.Float64frombits(0x40026bb1bbb55515)
	// Exceptional values cannot arise from the audited model's finite controls.
	if math.IsNaN(z) || math.IsInf(z, 0) || z > 88.72283935546875 || z <= -103.2789306640625 {
		return float32(math.Pow(10, float64(exponent)))
	}
	q := int64(math.RoundToEven(z * math.Float64frombits(0x40571547652b82fe)))
	r := math.FMA(-float64(q), math.Float64frombits(0x3f862e42fefa39ef), z)
	p := math.FMA(math.Float64frombits(0x3fc5555555555555), r, 0.5)
	p = math.FMA(p, r, 1)
	u := r * p
	scale := math.Float64frombits(originalMagnitudeExpTable[q&63])
	value := math.FMA(u, scale, scale)
	value = math.Float64frombits(math.Float64bits(value) + (uint64(q>>6) << 52))
	return float32(value)
}

// Exact binary64 constants of the original powf 1/64 exponent table.
var originalMagnitudeExpTable = [...]uint64{
	0x3ff0000000000000, 0x3ff02c9a3e778061, 0x3ff059b0d3158574, 0x3ff0874518759bc8,
	0x3ff0b5586cf9890f, 0x3ff0e3ec32d3d1a2, 0x3ff11301d0125b51, 0x3ff1429aaea92de0,
	0x3ff172b83c7d517b, 0x3ff1a35beb6fcb75, 0x3ff1d4873168b9aa, 0x3ff2063b88628cd6,
	0x3ff2387a6e756238, 0x3ff26b4565e27cdd, 0x3ff29e9df51fdee1, 0x3ff2d285a6e4030b,
	0x3ff306fe0a31b715, 0x3ff33c08b26416ff, 0x3ff371a7373aa9cb, 0x3ff3a7db34e59ff7,
	0x3ff3dea64c123422, 0x3ff4160a21f72e2a, 0x3ff44e086061892d, 0x3ff486a2b5c13cd0,
	0x3ff4bfdad5362a27, 0x3ff4f9b2769d2ca7, 0x3ff5342b569d4f82, 0x3ff56f4736b527da,
	0x3ff5ab07dd485429, 0x3ff5e76f15ad2148, 0x3ff6247eb03a5585, 0x3ff6623882552225,
	0x3ff6a09e667f3bcd, 0x3ff6dfb23c651a2f, 0x3ff71f75e8ec5f74, 0x3ff75feb564267c9,
	0x3ff7a11473eb0187, 0x3ff7e2f336cf4e62, 0x3ff82589994cce13, 0x3ff868d99b4492ed,
	0x3ff8ace5422aa0db, 0x3ff8f1ae99157736, 0x3ff93737b0cdc5e5, 0x3ff97d829fde4e50,
	0x3ff9c49182a3f090, 0x3ffa0c667b5de565, 0x3ffa5503b23e255d, 0x3ffa9e6b5579fdbf,
	0x3ffae89f995ad3ad, 0x3ffb33a2b84f15fb, 0x3ffb7f76f2fb5e47, 0x3ffbcc1e904bc1d2,
	0x3ffc199bdd85529c, 0x3ffc67f12e57d14b, 0x3ffcb720dcef9069, 0x3ffd072d4a07897c,
	0x3ffd5818dcfba487, 0x3ffda9e603db3285, 0x3ffdfc97337b9b5f, 0x3ffe502ee78b3ff6,
	0x3ffea4afa2a490da, 0x3ffefa1bee615a27, 0x3fff50765b6e4540, 0x3fffa7c1819e90d8,
}
