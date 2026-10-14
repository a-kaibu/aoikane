package onnx

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"
)

func TestBoundLayersMatchOriginalModel(t *testing.T) {
	b, err := OpenBoundLayers("../../assets/boundmodel/bou.onnx")
	if err != nil {
		t.Fatal(err)
	}
	r := rand.New(rand.NewPCG(3, 7))
	hash := sha256.New()
	classifierHash := sha256.New()
	count := 0
	for _, steps := range []int{1, 2, 5, 17, 33, 65, 257} {
		for _, scale := range []float32{0, 0.125, 1, 4} {
			input := make([]float32, steps*233)
			for i := range input {
				input[i] = float32(r.Float64()*2-1) * scale
			}
			first, second, err := b.Run(input, steps)
			if err != nil {
				t.Fatal(err)
			}
			for _, output := range [][]float32{first, second} {
				if len(output) != steps*128 {
					t.Fatal("LSTM output dimensions differ")
				}
				var raw []byte
				for _, x := range output {
					raw = binary.LittleEndian.AppendUint32(raw, math.Float32bits(x))
				}
				hash.Write(raw)
				count += len(output)
			}
			prob, err := b.Predict(input, steps)
			if err != nil {
				t.Fatal(err)
			}
			if len(prob) != steps*3 {
				t.Fatal("classifier output dimensions differ")
			}
			var raw []byte
			for _, x := range prob {
				raw = binary.LittleEndian.AppendUint32(raw, math.Float32bits(x))
			}
			classifierHash.Write(raw)
		}
	}
	// Original ONNX Runtime output, recorded by verify_bound_layers before
	// fixing this golden. Both directions/layers and every step are included.
	const want = "b75845db66ff3190e3e31ec4fc657e4194795efecb6b12b594b22c3983c1bced"
	if got := fmt.Sprintf("%x", hash.Sum(nil)); got != want {
		t.Fatalf("%d LSTM values: SHA256 %s want %s", count, got, want)
	}
	const classifierWant = "cb07bb65922055c39b7808278de0cd21ee909a45c556f5c1a643370d92a650d0"
	if got := fmt.Sprintf("%x", classifierHash.Sum(nil)); got != classifierWant {
		t.Fatalf("4560 original bou values: SHA256 %s want %s", got, classifierWant)
	}
	for _, bad := range []struct {
		input []float32
		steps int
	}{{nil, 0}, {make([]float32, 233), 0}, {make([]float32, 234), 1}, {make([]float32, 233), 2}} {
		if _, _, err := b.Run(bad.input, bad.steps); err == nil {
			t.Fatal("invalid feature dimensions accepted")
		}
	}
}

func TestBoundLayersRejectUnauditedGraph(t *testing.T) {
	data, err := os.ReadFile("../../assets/boundmodel/bou.onnx")
	if err != nil {
		t.Fatal(err)
	}
	data[len(data)-1] ^= 1
	path := filepath.Join(t.TempDir(), "modified.onnx")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenBoundLayers(path); err == nil {
		t.Fatal("modified model accepted by specialized executor")
	}
}
