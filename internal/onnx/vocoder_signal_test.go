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

func TestOriginalVocoderSignal(t *testing.T) {
	for _, tc := range []struct {
		name, hash  string
		count, call int
	}{
		{"vocoder-signal-native", "8cee4b95d0a87be77dafc1cde486b029a77626d5c23b3b5a91cd3e842b7f23c9", 52800, 1},
		{"vocoder-signal-short", "5dabfcc6468610766208eabb1182f4e68ee88c97cb5471f4427115a0b04045fb", 36000, 1},
		{"vocoder-signal-high", "d126bddce840e018a1be633d5c8ef9757dda8b3b1d02ed75adede82e1aacaeb4", 52800, 1},
		{"vocoder-signal-mixed-safe", "33b7727438fc37dccd51a406b8c04cea47742d2a70ffacc4886397339e6a4de9", 52800, 1},
		{"vocoder-signal-middle", "75991912f054114322aba9ab66994a9fe98daef319ffc6c7a36c08a3190140b6", 26400, 2},
		{"vocoder-signal-block", "62cb6743f4e32e464e432f37f09b86d4522e8ddd7f71ae9f0465d7eb220cec6a", 52800, 2},
		{"vocoder-signal-zero", "1a679f867090ec582b3af6b96aaf03ac1d6a1ae0e7badf50e5debecffe7c49fd", 52800, 1},
	} {
		t.Run(tc.name, func(t *testing.T) { testOriginalVocoderSignal(t, tc.name, tc.hash, tc.count, tc.call) })
	}
}

func testOriginalVocoderSignal(t *testing.T, name, hash string, count, call int) {
	data, err := os.ReadFile("../../private/" + name + "/signal.json")
	if os.IsNotExist(err) {
		t.Skip("private original excitation unavailable")
	}
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		ModelHash  string `json:"model_sha256"`
		ScriptHash string `json:"torchscript_sha256"`
		DLLHash    string `json:"dll_sha256"`
		Call       int    `json:"call_index"`
		F0         phaseTensor
		Means      []struct{ Input, Output phaseTensor }
		Signal     phaseTensor
		Hash       string `json:"signal_sha256"`
	}
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.ModelHash != "ff245a6149817c01c2d559e49087892623116766d1f3c95e7996bac23e6887e0" || ref.ScriptHash != "f63fe6b24c0de3cac10c6720adc3002166f1856493d8f91105ca77c4e6f45737" || ref.DLLHash != "00ca5c937940a94d052dbe52ec5fb0909db11ffbfccfd6e65e19b468169ff5e5" || ref.Hash != hash || len(ref.Signal.Data) != count || len(ref.Means) != 2 || ref.Call != call {
		t.Fatal("original excitation provenance/count/hash differs")
	}
	for _, m := range ref.Means {
		if !slices.Equal(m.Input.Shape, ref.F0.Shape) || !slices.Equal(m.Input.Data, ref.F0.Data) || len(m.Output.Shape) != 0 {
			t.Fatal("original global mean input/shape differs")
		}
		got, err := torchMeanAll(m.Input.Data)
		if err != nil {
			t.Fatal(err)
		}
		if len(m.Output.Data) != 1 || math.Float32bits(got) != math.Float32bits(m.Output.Data[0]) {
			t.Fatalf("original global mean got %.9g want %v", got, m.Output.Data)
		}
	}
	g, err := OpenGraph("../../assets/synth/vocoder.onnx")
	if err != nil {
		t.Fatal(err)
	}
	got, err := g.RunOriginalKernels(map[string]Value{"f0": {Kind: 1, Shape: ref.F0.Shape, F: ref.F0.Data}}, []string{"/Concat_4_output_0"})
	if err != nil {
		t.Fatal(err)
	}
	v := got["/Concat_4_output_0"]
	if !slices.Equal(v.Shape, ref.Signal.Shape) {
		t.Fatal("original excitation shape differs")
	}
	if len(v.F) != len(ref.Signal.Data) {
		t.Fatalf("excitation size differs %d %d", len(v.F), len(ref.Signal.Data))
	}
	h := sha256.New()
	for i, x := range v.F {
		if math.Float32bits(x) != math.Float32bits(ref.Signal.Data[i]) {
			t.Fatalf("original excitation %d got %08x want %08x", i, math.Float32bits(x), math.Float32bits(ref.Signal.Data[i]))
		}
		h.Write(binary.LittleEndian.AppendUint32(nil, math.Float32bits(x)))
	}
	if fmt.Sprintf("%x", h.Sum(nil)) != ref.Hash {
		t.Fatal("original excitation hash differs")
	}
	t.Logf("Original excitation %d values SHA256 %x", len(v.F), h.Sum(nil))
}
