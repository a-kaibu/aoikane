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

type phaseTensor struct {
	Shape []int64
	Data  []float32
}

func TestOriginalVocoderPrefix(t *testing.T) {
	for _, tc := range []struct {
		name, hash           string
		count, upsampleCount int
	}{
		{"vocoder-prefix-native", "4bebe6ef2902c29ce074091a3269f39847405de9e4f168fc054830888da88476", 158400, 101750},
		{"vocoder-prefix-short", "471a23df5e32f5bf868e346adafcfb3770df082906bcf41f610cfd2d16adf611", 108000, 69375},
		{"vocoder-sin-high", "c019a830e2a04671ba32c4605024775e9b9f70639c624ba3ec2ccf1b49020d86", 158400, 101750},
		{"vocoder-sin-large", "27c277a9b6ce539e168b5497d9b6349e1633d93d485ade9d2cd3c0d7ae36c723", 158400, 101750},
	} {
		t.Run(tc.name, func(t *testing.T) { testOriginalVocoderPrefix(t, tc.name, tc.hash, tc.count, tc.upsampleCount) })
	}
}

func testOriginalVocoderPrefix(t *testing.T, name, expectedHash string, expectedCount, upsampleCount int) {
	data, err := os.ReadFile("../../private/" + name + "/prefix.json")
	if os.IsNotExist(err) {
		t.Skip("private original vocoder prefix unavailable")
	}
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		ModelHash  string `json:"model_sha256"`
		ScriptHash string `json:"torchscript_sha256"`
		DLLHash    string `json:"dll_sha256"`
		Hash       string `json:"output_sha256"`
		F0         phaseTensor
		Prefix     []struct {
			Name string
			phaseTensor
		}
		Upsamples []struct{ Input, Output phaseTensor }
	}
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.ModelHash != "ff245a6149817c01c2d559e49087892623116766d1f3c95e7996bac23e6887e0" || ref.ScriptHash != "f63fe6b24c0de3cac10c6720adc3002166f1856493d8f91105ca77c4e6f45737" || ref.DLLHash != "00ca5c937940a94d052dbe52ec5fb0909db11ffbfccfd6e65e19b468169ff5e5" || len(ref.Prefix) != 6 || len(ref.Upsamples) != 7 {
		t.Fatal("unexpected original prefix provenance/count")
	}
	graph, err := OpenGraph("../../assets/synth/vocoder.onnx")
	if err != nil {
		t.Fatal(err)
	}
	outputs := make([]string, len(ref.Prefix))
	for i, p := range ref.Prefix {
		outputs[i] = p.Name
	}
	got, err := graph.RunOriginalKernels(map[string]Value{"f0": {Kind: 1, Shape: ref.F0.Shape, F: ref.F0.Data}}, outputs)
	if err != nil {
		t.Fatal(err)
	}
	hash, total := sha256.New(), 0
	largeAngles := 0
	for _, p := range ref.Prefix {
		value := got[p.Name]
		if value.Kind != 1 || !slices.Equal(value.Shape, p.Shape) || len(value.F) != len(p.Data) {
			t.Fatalf("%s shape/type differs", p.Name)
		}
		diffs := 0
		for i, x := range value.F {
			if p.Name == "/Mul_4_output_0" && math.Abs(float64(x)) > 10000 {
				largeAngles++
			}
			want := math.Float32bits(p.Data[i])
			if math.Float32bits(x) != want {
				diffs++
			}
			hash.Write(binary.LittleEndian.AppendUint32(nil, want))
		}
		t.Logf("%s values=%d bit_differences=%d", p.Name, len(value.F), diffs)
		if diffs != 0 {
			t.Errorf("%s differs from original in %d values", p.Name, diffs)
		}
		total += len(p.Data)
	}
	if name == "vocoder-sin-large" && largeAngles == 0 {
		t.Fatal("original high-F0 case did not exercise large Sin arguments")
	}
	t.Logf("Original phase angles above 10000: %d", largeAngles)
	if total != expectedCount || ref.Hash != expectedHash || fmt.Sprintf("%x", hash.Sum(nil)) != ref.Hash {
		t.Fatal("original prefix count/hash differs")
	}
	upsampled := 0
	for i, r := range ref.Upsamples {
		x := Value{Kind: 1, Shape: r.Input.Shape, F: r.Input.Data}
		node := graphNode{op: "Resize", attrs: map[string]graphAttr{"mode": {text: "nearest"}, "nearest_mode": {text: "floor"}, "coordinate_transformation_mode": {text: "asymmetric"}}}
		y, err := graphResizeNearest1D(node, []Value{x, {}, {}, {Kind: 7, Shape: []int64{3}, I: r.Output.Shape}})
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(y[0].Shape, r.Output.Shape) || len(y[0].F) != len(r.Output.Data) {
			t.Fatal("original upsample shape differs")
		}
		for j, v := range y[0].F {
			if math.Float32bits(v) != math.Float32bits(r.Output.Data[j]) {
				t.Fatalf("upsample %d value %d differs", i, j)
			}
		}
		upsampled += len(y[0].F)
	}
	if upsampled != upsampleCount {
		t.Fatal("original upsample value count differs")
	}
}
