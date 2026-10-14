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

func TestOriginalVocoderConvolutions(t *testing.T) {
	for _, tc := range []struct {
		name, hash  string
		count, call int
	}{
		{"native", "05111362e23a5190765aa3e3e3f5eccfb4cefa9fb2129dbe770f5b550517181f", 872960, 1},
		{"short", "9fef5de89b019aa30d6d1e5304e656fc7c3f248a7f3d25441c133ac1313817c3", 595200, 1},
		{"high", "11dc3260ad81ba5509bcbc0be98b61c5b2310aa57ec5dcc8a4545b198c0da7fa", 872960, 1},
		{"mixed", "787808eda71d94e8aae345cc5b3b235ef90782f272b38c3059083e4b9df4d1cd", 872960, 1},
		{"zero", "8d2107b44360d61650912cf3d223caf8f56c99d3ab5142b26620c201a68d876c", 872960, 1},
		{"tail", "35676b88b791c9250ed82528438f8669d3644ba9a386c4560acdf3af715f6f28", 436480, 2},
		{"block", "ed3187ca4cd9da892f2826e2f90dd0ef1f361dc3b582c62c6a655a16c469c01f", 872960, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			testOriginalVocoderConvolutions(t, "vocoder-downsample-"+tc.name, tc.hash, tc.count, tc.call, 6)
		})
	}
}

func TestOriginalVocoderFirstTranspose(t *testing.T) {
	for _, tc := range []struct {
		name, hash  string
		count, call int
	}{
		{"values", "0e8c74d3105ccff7c1363a26599d5131641ac328351fff2cc6d219213291fcc3", 943360, 1},
		{"short", "a77b5b0aaae2b0cd2231f6487082817c0d3fbb9c640c61a5f6d225c730e57528", 643200, 1},
		{"tail", "0a974c1b7f9aafbf84f717d9418485723ee5af4a2c3065bfa7a359c1ddd6bcba", 471680, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			testOriginalVocoderConvolutions(t, "vocoder-transpose-"+tc.name, tc.hash, tc.count, tc.call, 7)
		})
	}
}

func TestOriginalVocoderAdaptiveBlock(t *testing.T) {
	for _, tc := range []struct {
		root, hash   string
		values, call int
	}{
		{"vocoder-all-conv-native", "83ca3de306d835a1d67a6f80758ada9c2467a72a910cc99bc9f600fd445ea064", 1224960, 1},
		{"vocoder-transpose-all-short", "f670f3968879789d27d9c94a164a54049374ebe79f88c429040ec1dda568ad12", 835200, 1},
		{"vocoder-transpose-all-tail", "f89aa230ca7fdeba5c0d1e2f8f1a9d8d638d52fc3f76392922769fea179f58c9", 612480, 2},
		{"vocoder-adaptive-high", "7506df550b75d7c4e48144a2ad8e6edf47924c1b08ff2ff3139e95a4a2ba6dda", 1224960, 1},
		{"vocoder-adaptive-mixed", "367ceb8433028fb3ae989599f8772206616e0a296de096a228923e42223ed6f5", 1224960, 1},
		{"vocoder-adaptive-zero", "e59bee3ae36953bf4a42aa62d3a3c8ac423399ad5a5bb61ccfdc8bcb93a98e96", 1224960, 1},
	} {
		t.Run(tc.root, func(t *testing.T) { testOriginalVocoderConvolutions(t, tc.root, tc.hash, tc.values, tc.call, 11) })
	}
}

func testOriginalVocoderConvolutions(t *testing.T, name, expectedHash string, expectedCount, expectedCall, layers int) {
	data, err := os.ReadFile("../../private/" + name + "/convolutions.json")
	if os.IsNotExist(err) {
		t.Skip("private original vocoder convolutions unavailable")
	}
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		ModelHash    string `json:"model_sha256"`
		ScriptHash   string `json:"torchscript_sha256"`
		DLLHash      string `json:"dll_sha256"`
		Hash         string `json:"convolution_sha256"`
		Call         int    `json:"call_index"`
		F0, Bap, Spc phaseTensor
		Convolutions []struct {
			Name                        string
			Input, Weight, Bias, Output phaseTensor
		}
	}
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.ModelHash != "ff245a6149817c01c2d559e49087892623116766d1f3c95e7996bac23e6887e0" || ref.ScriptHash != "f63fe6b24c0de3cac10c6720adc3002166f1856493d8f91105ca77c4e6f45737" || ref.DLLHash != "00ca5c937940a94d052dbe52ec5fb0909db11ffbfccfd6e65e19b468169ff5e5" || ref.Hash != expectedHash || len(ref.Convolutions) != layers || ref.Call != expectedCall {
		t.Fatal("original vocoder convolution provenance/count differs")
	}
	g, err := OpenGraph("../../assets/synth/vocoder.onnx")
	if err != nil {
		t.Fatal(err)
	}
	inputs := map[string]Value{}
	for name, v := range map[string]phaseTensor{"f0": ref.F0, "bap": ref.Bap, "spc": ref.Spc} {
		inputs[name] = Value{Kind: 1, Shape: v.Shape, F: v.Data}
	}
	var outputs []string
	for i, c := range ref.Convolutions {
		want := "/Conv_output_0"
		if i != 0 {
			want = fmt.Sprintf("/Conv_%d_output_0", i)
		}
		if i == 6 {
			want = "/ConvTranspose_output_0"
		} else if i > 6 {
			want = fmt.Sprintf("/Conv_%d_output_0", i-1)
		}
		if c.Name != want {
			t.Fatal("original convolution order differs")
		}
		n := g.nodes[g.producer[c.Name]]
		wantOp := "Conv"
		if i == 6 {
			wantOp = "ConvTranspose"
		}
		if n.op != wantOp || len(n.inputs) != 3 {
			t.Fatal("original convolution node differs")
		}
		outputs = append(outputs, c.Name)
		outputs = append(outputs, n.inputs...)
	}
	got, err := g.RunOriginalKernels(inputs, outputs)
	if err != nil {
		t.Fatal(err)
	}
	compare := func(name string, v Value, w phaseTensor) {
		if v.Kind != 1 || !slices.Equal(v.Shape, w.Shape) || len(v.F) != len(w.Data) {
			t.Fatalf("%s shape/count differs", name)
		}
		for i, x := range v.F {
			if math.Float32bits(x) != math.Float32bits(w.Data[i]) {
				t.Fatalf("%s input/parameter bit difference at %d", name, i)
			}
		}
	}
	h, total := sha256.New(), 0
	for _, c := range ref.Convolutions {
		n := g.nodes[g.producer[c.Name]]
		compare(n.inputs[0], got[n.inputs[0]], c.Input)
		compare(n.inputs[1], got[n.inputs[1]], c.Weight)
		compare(n.inputs[2], got[n.inputs[2]], c.Bias)
		v := got[c.Name]
		if !slices.Equal(v.Shape, c.Output.Shape) || len(v.F) != len(c.Output.Data) {
			t.Fatal("output count differs")
		}
		differences := 0
		var maxError float64
		for i, x := range v.F {
			h.Write(binary.LittleEndian.AppendUint32(nil, math.Float32bits(c.Output.Data[i])))
			total++
			if math.Float32bits(x) != math.Float32bits(c.Output.Data[i]) {
				differences++
				maxError = max(maxError, math.Abs(float64(x)-float64(c.Output.Data[i])))
			}
		}
		t.Logf("%s values=%d bit_differences=%d max_error=%g", c.Name, len(v.F), differences, maxError)
		if differences != 0 {
			t.Fail()
		}
	}
	if total != expectedCount || fmt.Sprintf("%x", h.Sum(nil)) != expectedHash {
		t.Fatal("original convolution total/hash differs")
	}
	t.Logf("Compared %d original continuous convolution values; SHA256 %x", total, h.Sum(nil))
}
