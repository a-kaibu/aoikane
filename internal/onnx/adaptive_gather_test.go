package onnx

import (
	"math"
	"runtime"
	"slices"
	"testing"
)

func TestAdaptiveGatherMatchesPaddedIndexing(t *testing.T) {
	const channels, frames, left, right = 3, 11, 3, 4
	x := Value{Kind: 1, Shape: []int64{1, channels, frames}, F: make([]float32, channels*frames)}
	for i := range x.F {
		x.F[i] = float32(i) - 10.5
	}
	x.F[4] = float32(math.Copysign(0, -1))
	idx := Value{Kind: 7, Shape: []int64{1, 1, frames}, I: []int64{-18, -17, -1, 0, 2, 3, 7, 13, 14, 17, 8}}
	pads := Value{Kind: 7, Shape: []int64{6}, I: []int64{0, 0, left, 0, 0, right}}
	args := []Value{x, idx, pads, {Kind: 1, F: []float32{0}}}
	got, ok := runAdaptiveGather(args)
	if !ok || !slices.Equal(got.Shape, x.Shape) {
		t.Fatal("valid gather rejected")
	}
	defer putFloats(got.F)
	for c := 0; c < channels; c++ {
		padded := make([]float32, frames+left+right)
		copy(padded[left:], x.F[c*frames:(c+1)*frames])
		for r, index := range idx.I {
			if index < 0 {
				index += int64(len(padded))
			}
			want := padded[index]
			if math.Float32bits(got.F[c*frames+r]) != math.Float32bits(want) {
				t.Fatalf("channel=%d row=%d got=%g want=%g", c, r, got.F[c*frames+r], want)
			}
		}
	}
	for _, test := range []struct {
		name   string
		change func([]Value)
	}{
		{"index before start", func(v []Value) { v[1].I[0] = -19 }},
		{"index after end", func(v []Value) { v[1].I[0] = 18 }},
		{"negative padding", func(v []Value) { v[2].I[2] = -1 }},
		{"channel padding", func(v []Value) { v[2].I[1] = 1 }},
		{"nonzero fill", func(v []Value) { v[3].F[0] = 1 }},
		{"negative zero fill", func(v []Value) { v[3].F[0] = float32(math.Copysign(0, -1)) }},
		{"non-matching index shape", func(v []Value) { v[1].Shape = []int64{frames} }},
		{"short data", func(v []Value) { v[0].F = v[0].F[:1] }},
	} {
		t.Run(test.name, func(t *testing.T) {
			v := make([]Value, len(args))
			for i, a := range args {
				v[i] = cloneValue(a)
			}
			test.change(v)
			if _, ok := runAdaptiveGather(v); ok {
				t.Fatal("invalid geometry accepted")
			}
		})
	}
}

func TestAdaptiveGatherAuditedModelPlan(t *testing.T) {
	if runtime.GOOS == "js" {
		t.Skip("audited model is verified by native and browser integration tests")
	}
	g, err := OpenGraph("../../assets/synth/vocoder.onnx")
	if err != nil {
		t.Fatal(err)
	}
	if len(g.adaptive) != 30 {
		t.Fatalf("got %d adaptive plans, want 30", len(g.adaptive))
	}
	for index, plan := range g.adaptive {
		if g.nodes[index].op != "Reshape" {
			t.Fatal("unexpected fusion root")
		}
		for _, name := range plan.inputs {
			if name == "" {
				t.Fatal("missing dependency")
			}
		}
	}
}

func TestAdaptiveGatherIntermediateOutputsKeepDependencies(t *testing.T) {
	const frames, channels = 2048, 4
	x := Value{Kind: 1, Shape: []int64{1, channels, frames}, F: make([]float32, frames*channels)}
	for i := range x.F {
		x.F[i] = float32(i) + 0.25
	}
	idx := Value{Kind: 7, Shape: []int64{1, 1, frames}, I: make([]int64, frames)}
	for i := range idx.I {
		idx.I[i] = int64(i)
	}
	g := &Graph{
		initial:  map[string]Value{"zero": {Kind: 1, F: []float32{0}}, "two": {Kind: 1, F: []float32{2}}, "pads": {Kind: 7, Shape: []int64{6}, I: make([]int64, 6)}, "shape": {Kind: 7, Shape: []int64{3}, I: []int64{1, channels, frames}}},
		producer: map[string]int{"source": 0, "pad": 1, "root": 2, "other": 3},
		nodes: []graphNode{
			{op: "Add", inputs: []string{"x", "zero"}, outputs: []string{"source"}},
			{op: "Pad", inputs: []string{"source", "pads", "zero"}, outputs: []string{"pad"}},
			{op: "Reshape", inputs: []string{"pad", "shape"}, outputs: []string{"root"}},
			{op: "Mul", inputs: []string{"root", "two"}, outputs: []string{"other"}},
		},
		adaptive: map[int]adaptiveGatherPlan{2: {inputs: [4]string{"source", "idx", "pads", "zero"}}},
	}
	// Multiplication can recycle a prematurely released source buffer before
	// the explicitly requested Pad is evaluated. This used to corrupt it.
	out, err := g.Run(map[string]Value{"x": x, "idx": idx}, []string{"other", "pad"})
	if err != nil {
		t.Fatal(err)
	}
	for i, v := range x.F {
		if out["pad"].F[i] != v || out["other"].F[i] != v*2 {
			t.Fatalf("buffer reuse corrupted element %d", i)
		}
	}
}
