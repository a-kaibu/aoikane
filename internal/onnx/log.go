// Copyright (C) 2007 Julien Pommier
// Copyright (C) 2014 Pedro Gonnet
// Copyright (C) 2009-2019 Gael Guennebaud
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. See EIGEN-LICENSE or https://mozilla.org/MPL/2.0/.

package onnx

import "math"

// logPacket implements Eigen e7248b26's plog_impl_float, using the separate
// float32 multiply and add of the bundled Runtime's SSE packet evaluation.
func logPacket(input float32) float32 {
	if input == 0 {
		return float32(math.Inf(-1))
	}
	if input < 0 || math.IsNaN(float64(input)) {
		return math.Float32frombits(0xffffffff)
	}
	if math.IsInf(float64(input), 1) {
		return input
	}
	x := input
	if x < 0x1p-126 {
		x = 0x1p-126
	}
	bits := math.Float32bits(x)
	e := float32(int(bits>>23) - 126)
	x = math.Float32frombits((bits & 0x7fffff) | 0x3f000000)
	tmp := float32(0)
	if x < float32(0.707106781186547524) {
		tmp = x
		e = float32(e - 1)
	}
	x = float32(x - 1)
	x = float32(x + tmp)
	x2 := float32(x * x)
	x3 := float32(x2 * x)
	madd := func(a, b, c float32) float32 { product := float32(a * b); return float32(product + c) }
	y := madd(7.0376836292e-2, x, -1.1514610310e-1)
	y1 := madd(-1.2420140846e-1, x, 1.4249322787e-1)
	y2 := madd(2.0000714765e-1, x, -2.4999993993e-1)
	y = madd(y, x, 1.1676998740e-1)
	y1 = madd(y1, x, -1.6668057665e-1)
	y2 = madd(y2, x, 3.3333331174e-1)
	y = madd(y, x3, y1)
	y = madd(y, x3, y2)
	y = float32(y * x3)
	y = madd(-0.5, x2, y)
	x = float32(x + y)
	return madd(e, 0.693147180559945309417232121458176568, x)
}

func logValues(input []float32) []float32 {
	out := make([]float32, len(input))
	packetEnd := len(input) &^ 3
	for i, x := range input {
		if i < packetEnd {
			out[i] = logPacket(x)
		} else {
			out[i] = float32(math.Log(float64(x)))
		}
	}
	return out
}
