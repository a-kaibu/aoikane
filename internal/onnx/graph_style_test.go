package onnx

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"testing"
)

func TestStyleConditionsMatchOriginal(t *testing.T) {
	testStyleGraphMatchesOriginal(t, false)
}

func TestStyleNormalizationMatchesOriginal(t *testing.T) {
	testStyleGraphMatchesOriginal(t, true)
}

func testStyleGraphMatchesOriginal(t *testing.T, normalization bool) {
	t.Helper()
	g, err := OpenGraph("../../assets/synth/featnet.onnx")
	if err != nil {
		t.Fatal(err)
	}
	var outputs []string
	for i, n := range g.nodes {
		if i >= 50 && i <= 66 && n.op != "Constant" {
			outputs = append(outputs, n.outputs[0])
		}
	}
	if normalization {
		outputs = append(outputs, g.nodes[67].outputs...)
		outputs = append(outputs, g.nodes[68].outputs...)
	}
	hash := sha256.New()
	count := 0
	for _, length := range []int{1, 2, 5, 17, 65, 257} {
		shapes := [][3]int{{1, 8, 1}, {1, 8, length}, {5, 8, 1}, {5, 8, length}, {1, 7, 1}, {2, 7, length}}
		if normalization {
			shapes = [][3]int{{1, 8, 1}, {1, 8, length}, {5, 8, 1}, {5, 8, length}, {5, 7, length}, {5, 8, length + 1}}
		}
		for _, shape := range shapes {
			inputs := embeddingInputs(length)
			inputs["sty"] = Value{Kind: 1, Shape: []int64{int64(shape[0]), int64(shape[1]), int64(shape[2])}, F: make([]float32, shape[0]*shape[1]*shape[2])}
			if normalization {
				v := inputs["sty"]
				for i := range v.F {
					v.F[i] = float32(i%13-6) / 8
				}
			}
			result, err := g.Run(inputs, outputs)
			if err != nil {
				t.Fatal(err)
			}
			for _, name := range outputs {
				v := result[name]
				var values []float32
				switch v.Kind {
				case 1:
					values = v.F
				case 7:
					for _, x := range v.I {
						values = append(values, float32(x))
					}
				case 9:
					for _, x := range v.B {
						f := float32(0)
						if x {
							f = 1
						}
						values = append(values, f)
					}
				default:
					t.Fatalf("unexpected kind %d", v.Kind)
				}
				for _, x := range values {
					hash.Write(binary.LittleEndian.AppendUint32(nil, math.Float32bits(x)))
				}
				count += len(values)
			}
		}
	}
	// Recorded independently by verify_feat_embedding -style-conditions from
	// the original ONNX Runtime, including true/false branches and shape values.
	want, wantCount := "8bae31ab6435b915fb5d5680c2ef9a25d80d347488e4dfcfbebc032bc04c9da7", 540
	if normalization {
		want = "62bebe800ea86afa9ab1d0856477dc5d0582e6dc38d2bb9801369dfc3ab20e65"
		wantCount = 164110
	}
	if got := fmt.Sprintf("%x", hash.Sum(nil)); got != want || count != wantCount {
		t.Fatalf("%d values hash %s want %s", count, got, want)
	}
}

func TestIfCapturesParentAndSkipsUnchosenBranch(t *testing.T) {
	good := &Graph{initial: map[string]Value{}, producer: map[string]int{"out": 0}, outputs: []string{"out"}, nodes: []graphNode{{op: "Identity", inputs: []string{"captured"}, outputs: []string{"out"}}}}
	bad := &Graph{initial: map[string]Value{}, producer: map[string]int{"out": 0}, outputs: []string{"out"}, nodes: []graphNode{{op: "Unsupported", outputs: []string{"out"}}}}
	for _, condition := range []bool{false, true} {
		then, otherwise := good, bad
		if !condition {
			then, otherwise = bad, good
		}
		g := &Graph{initial: map[string]Value{}, producer: map[string]int{"result": 0}, nodes: []graphNode{{op: "If", inputs: []string{"condition"}, outputs: []string{"result"}, attrs: map[string]graphAttr{"then_branch": {graph: then}, "else_branch": {graph: otherwise}}}}}
		out, err := g.Run(map[string]Value{"condition": {Kind: 9, B: []bool{condition}}, "captured": {Kind: 7, I: []int64{37}}}, []string{"result"})
		if err != nil {
			t.Fatal(err)
		}
		if out["result"].I[0] != 37 {
			t.Fatal("parent capture differs")
		}
		if _, err := g.Run(map[string]Value{"condition": {Kind: 9, B: []bool{!condition}}}, []string{"result"}); err == nil {
			t.Fatal("selected unsupported branch accepted")
		}
	}
}
