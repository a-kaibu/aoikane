package onnx

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand/v2"
	"testing"
)

func TestPauseModelMatchesOriginal(t *testing.T) {
	p, err := OpenPauseModel("../../assets/boundmodel/pau.onnx")
	if err != nil {
		t.Fatal(err)
	}
	r := rand.New(rand.NewPCG(3, 7))
	layersHash, outputHash := sha256.New(), sha256.New()
	for _, steps := range []int{1, 2, 5, 17} {
		for _, scale := range []float32{0, 0.125, 1, 4} {
			input := make([]float32, steps*233)
			for i := range input {
				input[i] = float32(r.Float64()*2-1) * scale
			}
			for variant := 0; variant < 7; variant++ {
				n := steps
				if variant == 5 {
					n = max(1, steps/2)
				}
				if variant == 6 {
					n = steps + 2
				}
				opt := make([]float32, n*3)
				for s := 0; s < n; s++ {
					best := variant
					if best >= 3 {
						best = s % 3
					}
					opt[s*3+best] = 0.7
					if variant == 4 {
						opt[s*3], opt[s*3+1], opt[s*3+2] = 1, 1, 1
					}
				}
				first, second, err := p.RunLayers(input, opt, steps)
				if err != nil {
					t.Fatal(err)
				}
				for _, v := range [][]float32{first, second} {
					if len(v) != steps*128 {
						t.Fatal("LSTM output shape differs")
					}
					var raw []byte
					for _, x := range v {
						raw = binary.LittleEndian.AppendUint32(raw, math.Float32bits(x))
					}
					layersHash.Write(raw)
				}
				out, err := p.Predict(input, opt, steps)
				if err != nil {
					t.Fatal(err)
				}
				if len(out) != steps*3 {
					t.Fatal("pause output shape differs")
				}
				var raw []byte
				for _, x := range out {
					raw = binary.LittleEndian.AppendUint32(raw, math.Float32bits(x))
				}
				outputHash.Write(raw)
			}
		}
	}
	// Recorded from the exposed intermediate graph and the unmodified pau
	// graph, both executed in original ONNX Runtime by verify_bound_layers.
	if got := fmt.Sprintf("%x", layersHash.Sum(nil)); got != "ca5fede23e29eeaa841759bc345ecac6a8e2032084bfdd7c4d3ef485aed05998" {
		t.Fatalf("179200 LSTM values differ: SHA256 %s", got)
	}
	if got := fmt.Sprintf("%x", outputHash.Sum(nil)); got != "37cb2aacff9960981ef7f55258462fa43b1bd1da408bc4a04370850ade251118" {
		t.Fatalf("2100 original pau values differ: SHA256 %s", got)
	}
}

func TestPauseRejectsInvalidInput(t *testing.T) {
	for _, tc := range []struct {
		features, opt []float32
		steps         int
	}{
		{nil, nil, 0}, {make([]float32, 233), nil, 1}, {make([]float32, 234), make([]float32, 3), 1}, {make([]float32, 233), make([]float32, 4), 1}, {make([]float32, 233), make([]float32, 3), 2},
	} {
		if _, err := pauseFeatures(tc.features, tc.opt, tc.steps); err == nil {
			t.Fatal("invalid pause inputs accepted")
		}
	}
	if _, err := OpenPauseModel("../../assets/boundmodel/bou.onnx"); err == nil {
		t.Fatal("bou accepted as a pause graph")
	}
}
