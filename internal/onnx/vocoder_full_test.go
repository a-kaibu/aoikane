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

type nativeVocoderTensor struct {
	ID            int
	Shape, Stride []int64
	Dtype         int
}

// Read logical values without assuming that a captured native view is contiguous.
func readNativeVocoderTensor(t *testing.T, root string, ref nativeVocoderTensor) Value {
	t.Helper()
	if ref.Dtype != 6 || len(ref.Shape) != len(ref.Stride) {
		t.Fatal("invalid native FLOAT metadata")
	}
	count, last := int64(1), int64(0)
	for i, d := range ref.Shape {
		if d < 1 || ref.Stride[i] < 0 || d > (1<<30)/count || ref.Stride[i] > (1<<30)/d {
			t.Fatal("invalid native tensor extent")
		}
		count *= d
		last += (d - 1) * ref.Stride[i]
	}
	raw, err := os.ReadFile(fmt.Sprintf("%s/model-%d.bin", root, ref.ID))
	if err != nil {
		t.Fatal(err)
	}
	if last >= int64(len(raw))/4 {
		t.Fatal("native tensor exceeds captured storage")
	}
	v := Value{Kind: 1, Shape: ref.Shape, F: make([]float32, count)}
	for i := range v.F {
		left, offset := int64(i), int64(0)
		for a := len(ref.Shape) - 1; a >= 0; a-- {
			offset += left % ref.Shape[a] * ref.Stride[a]
			left /= ref.Shape[a]
		}
		v.F[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[offset*4:]))
	}
	return v
}

// Continuous original-input execution of the whole vocoder, including Tanh.
// Native intermediate values are references only, never graph inputs.
func TestOriginalVocoderWholeGraph(t *testing.T) {
	for _, tc := range []struct {
		root, hash, waveHash string
		count, call, threads int
	}{
		{"thread1-kan-vocoder-ops", "bd5d13052e3b1704ff06b183cef18c315187a980673874c0b93bd599ef59c675", "e8f3e182663de545a0b9f81adcc12da38e61aee695393d4df690b421a15c5ede", 43264206, 1, 1},
		{"thread1-vocoder-ops", "3947845a74fb331791734d6767fa55997655de6bf5b581a8fac87db925e9b640", "9994cc7f0997e02bbec2281ca8c527ec45657a3b0ee030378ff798ea1a8a4ea8", 46918503, 1, 1},
		{"vocoder-all-conv-native", "154534c304e20af6d73d3969d3bd51efc874bd952d1fd1924e036c5b08c53728", "52b6151ac4b02d6a5144b6d19beedb38e45f7c0c4250f25e80ff83a70c1f8238", 46918503, 1, 6},
		{"vocoder-transpose-all-short", "3a8363aae213138ae02d315ad3b23f1aaf92324285c0556e6306630649552d8d", "f0e47b656610fa8375abf26a314694d975043137c84039e4704f7a253b127300", 32707348, 1, 6},
		{"vocoder-transpose-all-tail", "cbb1ab3b12cf8f10105ed51f7ca6e522c6ddcc34ef762707c802a8216ec77918", "e8e4738a57c3b2999f50ee576e89ffc4aeb59d3e64efe0ba31643cb905ac173d", 24586688, 2, 6},
	} {
		t.Run(tc.root, func(t *testing.T) {
			root := "../../private/" + tc.root
			data, err := os.ReadFile(root + "/operators.json")
			if os.IsNotExist(err) {
				t.Skip("private original vocoder operator capture unavailable")
			}
			if err != nil {
				t.Fatal(err)
			}
			var ref struct {
				Wave       nativeVocoderTensor
				WaveHash   string `json:"native_wave_sha256"`
				ModelHash  string `json:"model_sha256"`
				ScriptHash string `json:"torchscript_sha256"`
				DLLHash    string `json:"dll_sha256"`
				Hash       string `json:"output_sha256"`
				Count      int    `json:"value_count"`
				Call       int    `json:"call_index"`
				Inputs     map[string]nativeVocoderTensor
				Operators  []struct {
					Node, Op string
					Names    []string
					Tensors  []nativeVocoderTensor
				}
			}
			if err = json.Unmarshal(data, &ref); err != nil {
				t.Fatal(err)
			}
			if ref.ModelHash != "ff245a6149817c01c2d559e49087892623116766d1f3c95e7996bac23e6887e0" || ref.ScriptHash != "f63fe6b24c0de3cac10c6720adc3002166f1856493d8f91105ca77c4e6f45737" || ref.DLLHash != "00ca5c937940a94d052dbe52ec5fb0909db11ffbfccfd6e65e19b468169ff5e5" || ref.WaveHash != tc.waveHash || ref.Hash != tc.hash || ref.Count != tc.count || ref.Call != tc.call || len(ref.Operators) != 126 || len(ref.Inputs) != 3 {
				t.Fatal("original vocoder provenance/count differs")
			}
			g, err := OpenGraph("../../assets/synth/vocoder.onnx")
			if err != nil {
				t.Fatal(err)
			}
			inputs := map[string]Value{}
			for _, name := range []string{"f0", "bap", "spc"} {
				inputs[name] = readNativeVocoderTensor(t, root, ref.Inputs[name])
			}
			var outputs []string
			seen := map[string]bool{}
			nodes := map[string]bool{}
			for _, r := range ref.Operators {
				if len(r.Names) != 4 || len(r.Tensors) != 4 || nodes[r.Names[3]] {
					t.Fatal("invalid/duplicate original operator")
				}
				nodes[r.Names[3]] = true
				index, ok := g.producer[r.Names[3]]
				if !ok {
					t.Fatal("original operator missing")
				}
				n := g.nodes[index]
				if n.op != r.Op || (n.op != "Conv" && n.op != "ConvTranspose") || !slices.Equal(n.inputs, r.Names[:3]) {
					t.Fatal("original operator mapping differs")
				}
				for _, name := range r.Names {
					if !seen[name] {
						seen[name] = true
						outputs = append(outputs, name)
					}
				}
			}
			outputs = append(outputs, "wave")
			got, err := g.RunOriginalKernelsWithThreads(inputs, outputs, tc.threads)
			if err != nil {
				t.Fatal(err)
			}
			h, total := sha256.New(), 0
			for _, r := range ref.Operators {
				// Fixed hash order: input, output, weight, bias.
				for _, j := range []int{0, 3, 1, 2} {
					name := r.Names[j]
					want := readNativeVocoderTensor(t, root, r.Tensors[j])
					v := got[name]
					if v.Kind != 1 || !slices.Equal(v.Shape, want.Shape) || len(v.F) != len(want.F) {
						t.Fatalf("%s shape/count differs", name)
					}
					var bits [4]byte
					for i, x := range v.F {
						w := math.Float32bits(want.F[i])
						if math.Float32bits(x) != w {
							t.Fatalf("%s original bit difference at %d: %08x != %08x", name, i, math.Float32bits(x), w)
						}
						binary.LittleEndian.PutUint32(bits[:], w)
						h.Write(bits[:])
					}
					total += len(v.F)
				}
			}
			if total != tc.count || fmt.Sprintf("%x", h.Sum(nil)) != tc.hash {
				t.Fatal("original vocoder total/hash differs")
			}
			wantWave := readNativeVocoderTensor(t, root, ref.Wave)
			wave := got["wave"]
			frames := inputs["f0"].Shape[2] * 240
			if wave.Kind != 1 || !slices.Equal(wave.Shape, []int64{1, 1, frames}) || !slices.Equal(wave.Shape, wantWave.Shape) || len(wave.F) != int(frames) || len(wave.F) != len(wantWave.F) {
				t.Fatal("original vocoder wave shape/count differs")
			}
			waveDigest := sha256.New()
			var bits [4]byte
			for i, x := range wave.F {
				if math.Float32bits(x) != math.Float32bits(wantWave.F[i]) {
					t.Fatalf("original wave bit difference at %d: %08x != %08x", i, math.Float32bits(x), math.Float32bits(wantWave.F[i]))
				}
				binary.LittleEndian.PutUint32(bits[:], math.Float32bits(x))
				waveDigest.Write(bits[:])
			}
			if fmt.Sprintf("%x", waveDigest.Sum(nil)) != tc.waveHash {
				t.Fatal("original wave hash differs")
			}
			t.Logf("Full vocoder wave: %d values, bit_differences=0, SHA256=%s", len(wave.F), tc.waveHash)
			t.Logf("126 continuous operators, %d input/output/weight/bias values, bit_differences=0, SHA256=%s", total, tc.hash)
		})
	}
}
