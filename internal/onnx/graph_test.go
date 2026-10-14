package onnx

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"testing"
)

func embeddingInputs(n int) map[string]Value {
	in := make(map[string]Value)
	for _, name := range []string{"phn", "art", "acc", "punc", "nima", "niab"} {
		v := Value{Kind: 1, Shape: []int64{1, int64(n)}}
		if name == "art" {
			v.Shape = []int64{1, 32, int64(n)}
		}
		if name == "phn" || name == "acc" || name == "punc" {
			v.Kind = 7
			v.I = make([]int64, n)
			for i := range v.I {
				mod := 52
				if name == "acc" {
					mod = 4
				}
				if name == "punc" {
					mod = 8
				}
				v.I[i] = int64(i % mod)
			}
		} else {
			count := n
			if name == "art" {
				count *= 32
			}
			v.F = make([]float32, count)
			for i := range v.F {
				switch name {
				case "art":
					v.F[i] = float32(i%5) / 4
				case "nima":
					v.F[i] = float32(i%13) - 2 + 0.25
				case "niab":
					v.F[i] = float32(i%7) - 1 + 0.5
				}
			}
		}
		in[name] = v
	}
	return in
}

func TestEmbeddingGraphMatchesOriginal(t *testing.T) {
	g, err := OpenGraph("../../assets/synth/featnet.onnx")
	if err != nil {
		t.Fatal(err)
	}
	var outputs []string
	for i, n := range g.nodes {
		if i >= 10 && i <= 49 && n.op != "Constant" {
			outputs = append(outputs, n.outputs[0])
		}
	}
	outputs = append(outputs, "/model/Unsqueeze_output_0")
	if len(outputs) != 32 {
		t.Fatalf("embedding output count %d", len(outputs))
	}
	hash := sha256.New()
	count := 0
	for _, length := range []int{1, 2, 5, 17, 65, 257} {
		out, err := g.Run(embeddingInputs(length), outputs)
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range outputs {
			v := out[name]
			values := v.F
			if v.Kind == 9 {
				values = make([]float32, len(v.B))
				for i, b := range v.B {
					if b {
						values[i] = 1
					}
				}
			}
			for _, x := range values {
				hash.Write(binary.LittleEndian.AppendUint32(nil, math.Float32bits(x)))
			}
			count += len(values)
		}
	}
	// All 32 original intermediates, independently recorded from original
	// Runtime by verify_feat_embedding -synthetic. Includes the padding mask.
	const want = "e874cc0e6f78403c19a6562eb4fc07a53645c31198e0471754796a5dcd5e0edb"
	if got := fmt.Sprintf("%x", hash.Sum(nil)); got != want {
		t.Fatalf("%d native embedding values: SHA256 %s want %s", count, got, want)
	}
	if _, err := g.Run(embeddingInputs(2), []string{"f0"}); err == nil {
		t.Fatal("F0 graph accepted missing style and control inputs")
	}
	if _, err := g.Run(nil, []string{outputs[0]}); err == nil {
		t.Fatal("missing phoneme input accepted")
	}
	bad := embeddingInputs(2)
	bad["art"] = Value{Kind: 1, Shape: []int64{1, 32, 2}, F: []float32{1}}
	if _, err := g.Run(bad, outputs); err == nil {
		t.Fatal("invalid tensor storage accepted")
	}
	for _, key := range []string{"model.emb_tok.blocks.phn.block.weight", outputs[0]} {
		if _, err := g.Run(map[string]Value{key: {Kind: 1, F: []float32{1}}}, []string{outputs[0]}); err == nil {
			t.Fatal("input overwrites model value")
		}
	}
	// Returned values must not mutate the immutable model or later calls.
	key := "model.emb_tok.blocks.phn.block.weight"
	v, err := g.Run(nil, []string{key})
	if err != nil {
		t.Fatal(err)
	}
	bits := math.Float32bits(v[key].F[0])
	v[key].F[0] = 99
	next, err := g.Run(nil, []string{key})
	if err != nil || math.Float32bits(next[key].F[0]) != bits {
		t.Fatal("returned tensor aliases model weights")
	}
}

func TestGraphRejectsCyclesAndIncompatibleBroadcast(t *testing.T) {
	g := &Graph{initial: map[string]Value{}, producer: map[string]int{"loop": 0}, nodes: []graphNode{{op: "Add", inputs: []string{"loop", "loop"}, outputs: []string{"loop"}}}}
	if _, err := g.Run(nil, []string{"loop"}); err == nil {
		t.Fatal("cyclic graph accepted")
	}
	if _, err := binaryValue("Add", Value{Kind: 1, Shape: []int64{2}, F: []float32{1, 2}}, Value{Kind: 1, Shape: []int64{3}, F: []float32{1, 2, 3}}); err == nil {
		t.Fatal("incompatible broadcast accepted")
	}
}
