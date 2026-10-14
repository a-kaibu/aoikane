package onnx

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"slices"
	"testing"
)

// These are isolated original-input comparisons for all ten ConvTranspose
// layers, including the two packed-GEMM layers. The full graph is incomplete.
func TestOriginalVocoderTransposed(t *testing.T) {
	for _, tc := range []struct {
		root, hash string
		values     int
	}{
		{"vocoder-all-conv-native", "6757cf093ba4f173a80dae64ccb92c7c9c2c4de93cf768f9dca363d7e9bca5b9", 1689600},
		{"vocoder-transpose-all-short", "6222299aac6667982c633faac1d8e7d0f3512bf6ab3283894395d593b4212a88", 1152000},
		{"vocoder-transpose-all-tail", "09940e7cf188e4387ab67d53a39d0e1f0d7fe8048c6a44146c56256846c6cd1b", 844800},
	} {
		t.Run(tc.root, func(t *testing.T) { testOriginalVocoderTransposed(t, tc.root, tc.hash, tc.values) })
	}
}
func testOriginalVocoderTransposed(t *testing.T, root, expectedHash string, expectedCount int) {
	data, err := os.ReadFile("../../private/" + root + "/transposed.json")
	if os.IsNotExist(err) {
		t.Skip("private original transposed convolutions unavailable")
	}
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		ModelHash    string `json:"model_sha256"`
		ScriptHash   string `json:"torchscript_sha256"`
		DLLHash      string `json:"dll_sha256"`
		Hash         string `json:"output_sha256"`
		Convolutions []struct {
			Name                        string
			InputName                   string `json:"input_name"`
			Input, Weight, Bias, Output phaseTensor
		}
	}
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.ModelHash != "ff245a6149817c01c2d559e49087892623116766d1f3c95e7996bac23e6887e0" || ref.ScriptHash != "f63fe6b24c0de3cac10c6720adc3002166f1856493d8f91105ca77c4e6f45737" || ref.DLLHash != "00ca5c937940a94d052dbe52ec5fb0909db11ffbfccfd6e65e19b468169ff5e5" || ref.Hash != expectedHash || len(ref.Convolutions) != 10 {
		t.Fatal("original transposed convolution provenance/count differs")
	}
	g, err := OpenGraph("../../assets/synth/vocoder.onnx")
	if err != nil {
		t.Fatal(err)
	}
	h, total := sha256.New(), 0
	seen := map[string]bool{}
	for _, c := range ref.Convolutions {
		index, ok := g.producer[c.Name]
		if !ok || seen[c.Name] {
			t.Fatal("missing/duplicate original transposed convolution")
		}
		seen[c.Name] = true
		n := g.nodes[index]
		if n.op != "ConvTranspose" || len(n.inputs) != 3 || n.inputs[0] != c.InputName {
			t.Fatal("original transposed convolution node differs")
		}
		input := Value{Kind: 1, Shape: c.Input.Shape, F: c.Input.Data}
		got, err := g.RunOriginalKernels(nil, []string{n.inputs[1], n.inputs[2]})
		if err != nil {
			t.Fatal(err)
		}
		out, err := executeNode(n, []Value{input, got[n.inputs[1]], got[n.inputs[2]]}, true)
		if err != nil {
			t.Fatal(err)
		}
		if len(out) != 1 {
			t.Fatal("original transposed convolution output count differs")
		}
		got[c.Name] = out[0]
		for name, w := range map[string]phaseTensor{c.Name: c.Output, n.inputs[1]: c.Weight, n.inputs[2]: c.Bias} {
			v := got[name]
			if v.Kind != 1 || !slices.Equal(v.Shape, w.Shape) || len(v.F) != len(w.Data) {
				t.Fatal("original transposed tensor shape differs")
			}
			for i, x := range v.F {
				if math.Float32bits(x) != math.Float32bits(w.Data[i]) {
					t.Fatalf("%s original bit difference at %d: %08x != %08x", name, i, math.Float32bits(x), math.Float32bits(w.Data[i]))
				}
			}
		}
		for _, x := range c.Output.Data {
			h.Write(binary.LittleEndian.AppendUint32(nil, math.Float32bits(x)))
			total++
		}
		t.Logf("%s stride=%d values=%d bit_differences=0", c.Name, n.attrs["strides"].ints[0], len(c.Output.Data))
	}
	if total != expectedCount || fmt.Sprintf("%x", h.Sum(nil)) != expectedHash {
		t.Fatal("original transposed convolution total/hash differs")
	}
	t.Logf("Compared %d original isolated transposed convolution values; SHA256 %x", total, h.Sum(nil))
}
