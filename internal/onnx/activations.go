package onnx

import "math"

// Coefficients and evaluation order are from ONNX Runtime 1.17.1 MLAS
// logistic.cpp/tanh.cpp (Microsoft, MIT; see MLAS-LICENSE). LSTM uses
// these float32 rational approximations rather than the Go float64 functions.

// fma32 rounds a*b+c once to float32, without depending on a host FMA
// instruction. A float32 product is exact in float64. Only a float64 result
// on a float32 rounding midpoint can need its exact addition residual.
func fma32(a, b, c float32) float32 {
	product := float64(a) * float64(b)
	sum := product + float64(c)
	result := float32(sum)
	abs := math.Abs(sum)
	if math.IsNaN(sum) || math.IsInf(sum, 0) {
		return float32(math.FMA(float64(a), float64(b), float64(c)))
	}
	if abs >= 0x1p-126 && abs < 0x1p128 && math.Float64bits(sum)&((1<<29)-1) != 1<<28 {
		return result
	}
	virtualC := sum - product
	residual := (product - (sum - virtualC)) + (float64(c) - virtualC)
	if residual == 0 {
		return result
	}
	if math.IsInf(float64(result), 0) {
		threshold := float64(math.MaxFloat32) + 0x1p103
		if sum == threshold && residual < 0 {
			return math.MaxFloat32
		}
		if sum == -threshold && residual > 0 {
			return -math.MaxFloat32
		}
		return result
	}
	direction := float32(math.Inf(1))
	if residual < 0 {
		direction = float32(math.Inf(-1))
	}
	next := math.Nextafter32(result, direction)
	if sum == (float64(result)+float64(next))*0.5 {
		return next
	}
	return result
}

func clamp(x, low, high float32) float32 {
	if x < low {
		return low
	}
	if x > high {
		return high
	}
	return x
}

func Sigmoid(x float32) float32 {
	x = clamp(x, -18, 18)
	y := float32(x * x)
	p := fma32(y, 4.37031012579801e-11, 1.15627324459942e-7)
	p = fma32(p, y, 6.08574864600143e-5)
	p = fma32(p, y, 8.51377133304701e-3)
	p = fma32(p, y, 2.48287947061529e-1)
	p = float32(p * x)
	q := fma32(y, 6.10247389755681e-13, 5.76102136993427e-9)
	q = fma32(q, y, 6.29106785017040e-6)
	q = fma32(q, y, 1.70198817374094e-3)
	q = fma32(q, y, 1.16817656904453e-1)
	q = fma32(q, y, 9.93151921023180e-1)
	result := float32(float32(p/q) + 0.5)
	// The FMA3 kernel also clamps a small polynomial undershoot to zero.
	if result < 0 {
		return 0
	}
	return result
}

func Tanh(x float32) float32 {
	x = clamp(x, -9, 9)
	y := float32(x * x)
	p := fma32(y, -2.76076847742355e-16, 2.00018790482477e-13)
	p = fma32(p, y, -8.60467152213735e-11)
	p = fma32(p, y, 5.12229709037114e-8)
	p = fma32(p, y, 1.48572235717979e-5)
	p = fma32(p, y, 6.37261928875436e-4)
	p = fma32(p, y, 4.89352455891786e-3)
	p = float32(p * x)
	q := fma32(y, 1.19825839466702e-6, 1.18534705686654e-4)
	q = fma32(q, y, 2.26843463243900e-3)
	q = fma32(q, y, 4.89352518554385e-3)
	return float32(p / q)
}
