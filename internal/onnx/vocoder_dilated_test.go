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

// These are isolated original-input comparisons. The adaptive/transpose
// operations before these layers are not yet executable as a continuous graph.
func TestOriginalVocoderDilated(t *testing.T) {
	data, err := os.ReadFile("../../private/vocoder-all-conv-native/dilated.json")
	if os.IsNotExist(err) {
		t.Skip("private original dilated convolutions unavailable")
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
	const expectedHash = "dc25905dbba25a299c6f331c76e567becce68ec56d8360cfb3a0436a1e5509e1"
	if ref.ModelHash != "ff245a6149817c01c2d559e49087892623116766d1f3c95e7996bac23e6887e0" || ref.ScriptHash != "f63fe6b24c0de3cac10c6720adc3002166f1856493d8f91105ca77c4e6f45737" || ref.DLLHash != "00ca5c937940a94d052dbe52ec5fb0909db11ffbfccfd6e65e19b468169ff5e5" || ref.Hash != expectedHash || len(ref.Convolutions) != 30 {
		t.Fatal("original dilated convolution provenance/count differs")
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
			t.Fatal("missing/duplicate original dilated convolution")
		}
		seen[c.Name] = true
		n := g.nodes[index]
		if n.op != "Conv" || len(n.inputs) != 3 || n.inputs[0] != c.InputName || len(n.attrs["dilations"].ints) != 1 || n.attrs["dilations"].ints[0] <= 1 {
			t.Fatal("original dilated convolution node differs")
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
			t.Fatal("original dilated convolution output count differs")
		}
		got[c.Name] = out[0]
		for name, w := range map[string]phaseTensor{c.Name: c.Output, n.inputs[1]: c.Weight, n.inputs[2]: c.Bias} {
			v := got[name]
			if v.Kind != 1 || !slices.Equal(v.Shape, w.Shape) || len(v.F) != len(w.Data) {
				t.Fatal("original dilated tensor shape differs")
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
		t.Logf("%s dilation=%d values=%d bit_differences=0", c.Name, n.attrs["dilations"].ints[0], len(c.Output.Data))
	}
	if total != 5068800 || fmt.Sprintf("%x", h.Sum(nil)) != expectedHash {
		t.Fatal("original dilated convolution total/hash differs")
	}
	t.Logf("Compared %d original isolated dilated convolution values; SHA256 %x", total, h.Sum(nil))
}
