package onnx

import (
	"math/rand"
	"testing"
)

// The FLOAT Pad rows must equal the generic element loop, which DOUBLE uses.
func TestPadFloatRowsMatchesGeneric(t *testing.T) {
	rng := rand.New(rand.NewSource(4))
	for trial := 0; trial < 500; trial++ {
		rank := 1 + rng.Intn(3)
		shape := make([]int64, rank)
		count := 1
		for i := range shape {
			shape[i] = int64(1 + rng.Intn(5))
			count *= int(shape[i])
		}
		pads := make([]int64, 2*rank)
		for i := range pads {
			pads[i] = int64(rng.Intn(4)) - 1
			if pads[i] < -shape[i%rank]/2 {
				pads[i] = 0
			}
		}
		f := Value{Kind: 1, Shape: shape, F: make([]float32, count)}
		d := Value{Kind: 11, Shape: shape, D: make([]float64, count)}
		for i := range f.F {
			f.F[i] = float32(rng.NormFloat64())
			d.D[i] = float64(f.F[i])
		}
		p := Value{Kind: 7, Shape: []int64{int64(2 * rank)}, I: pads}
		n := graphNode{op: "Pad", attrs: map[string]graphAttr{}}
		fo, _, ferr := executeSequenceNode(n, []Value{f, p}, true)
		do, _, derr := executeSequenceNode(n, []Value{d, p}, true)
		if (ferr == nil) != (derr == nil) {
			t.Fatalf("error mismatch %v %v", ferr, derr)
		}
		if ferr != nil {
			continue
		}
		for i, v := range fo[0].F {
			if float64(v) != do[0].D[i] {
				t.Fatalf("trial %d shape %v pads %v: element %d differs", trial, shape, pads, i)
			}
		}
	}
}
