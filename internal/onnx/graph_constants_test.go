package onnx

import "testing"

func TestGraphConstantNodesAreImmutableInitializers(t *testing.T) {
	g, err := OpenGraph("../../assets/synth/vocoder.onnx")
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, node := range g.nodes {
		if node.op != "Constant" {
			continue
		}
		name := node.outputs[0]
		v, ok := g.initial[name]
		if !ok {
			t.Fatalf("constant %s was not promoted", name)
		}
		if _, ok := g.producer[name]; ok {
			t.Fatalf("constant %s still requires execution", name)
		}
		if len(v.F) > 0 {
			if !isConstant(v.F) {
				t.Fatalf("constant %s is not eligible for packing caches", name)
			}
			first := v.F[0]
			out, err := g.Run(nil, []string{name})
			if err != nil {
				t.Fatal(err)
			}
			out[name].F[0] = 12345
			if g.initial[name].F[0] != first {
				t.Fatal("output aliases immutable constant")
			}
			checked++
			if checked == 8 {
				break
			}
		}
	}
	if checked != 8 {
		t.Fatalf("checked %d FLOAT constants, want 8", checked)
	}
}
