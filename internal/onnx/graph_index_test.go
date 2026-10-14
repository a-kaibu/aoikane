package onnx

import (
	"math"
	"testing"
)

func TestScatterNDSlicesAndBounds(t *testing.T) {
	data := Value{Kind: 1, Shape: []int64{2, 3}, F: []float32{1, 2, 3, 4, 5, 6}}
	update := Value{Kind: 1, Shape: []int64{1, 3}, F: []float32{math.Float32frombits(0x80000000), math.Float32frombits(0x7fc01234), 7}}
	index := Value{Kind: 7, Shape: []int64{1, 1}, I: []int64{-1}}
	out, e := executeNode(graphNode{op: "ScatterND"}, []Value{data, index, update}, true)
	if e != nil {
		t.Fatal(e)
	}
	for i := range update.F {
		if math.Float32bits(out[0].F[3+i]) != math.Float32bits(update.F[i]) {
			t.Fatal("ScatterND changed stored float bits")
		}
	}
	if data.F[3] != 4 || out[0].F[0] != 1 {
		t.Fatal("ScatterND mutated input or unselected slice")
	}
	for _, bad := range []int64{-3, 2} {
		index.I[0] = bad
		if _, e = executeNode(graphNode{op: "ScatterND"}, []Value{data, index, update}, true); e == nil {
			t.Fatal("out-of-range ScatterND index accepted")
		}
	}
	index.I[0] = 0
	if _, e = executeNode(graphNode{op: "ScatterND", attrs: map[string]graphAttr{"reduction": {text: "add"}}}, []Value{data, index, update}, true); e == nil {
		t.Fatal("unimplemented ScatterND reduction accepted")
	}
	update.Shape = []int64{3}
	if _, e = executeNode(graphNode{op: "ScatterND"}, []Value{data, index, update}, true); e == nil {
		t.Fatal("invalid ScatterND updates shape accepted")
	}
}

func TestFloatRangeRejectsInexactIntegers(t *testing.T) {
	scalar := func(x float32) Value { return Value{Kind: 1, F: []float32{x}} }
	for _, v := range [][3]float32{{0, 2, 0}, {0, 2, 0.5}, {0.5, 2, 1}, {0, 16777218, 1}} {
		if _, e := graphIntegerRange(graphNode{}, []Value{scalar(v[0]), scalar(v[1]), scalar(v[2])}); e == nil {
			t.Fatalf("unsupported float range accepted: %v", v)
		}
	}
}
