package onnx

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"testing"
)

func TestOriginalAVX512TransposeSamples(t *testing.T) {
	var ref struct {
		Operation int
		Samples   []struct {
			Channel, SourceFrame, Position int
			Input, Weight                  []uint32
			Bias, Output                   uint32
		}
	}
	raw, err := os.ReadFile("testdata/dnnl_avx512_transpose.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.Operation != 21 || len(ref.Samples) != 3 {
		t.Fatal("invalid original transpose reference")
	}
	for _, s := range ref.Samples {
		if len(s.Input) != 128*2 || len(s.Weight) != 128*8 {
			t.Fatal("invalid original sample extent")
		}
		x, w := make([]float32, len(s.Input)), make([]float32, len(s.Weight))
		for i, v := range s.Input {
			x[i] = math.Float32frombits(v)
		}
		for i, v := range s.Weight {
			w[i] = math.Float32frombits(v)
		}
		// Select one independent output lane and its two contributing input
		// frames. The native stride/padding phase is preserved by Position.
		y := originalAVX512TransposeDirect(x, w, []float32{math.Float32frombits(s.Bias)}, 128, 1, 8, 2, 8, 4, 2)
		if got := math.Float32bits(y[s.Position]); got != s.Output {
			t.Fatalf("channel=%d frame=%d: got %08x want %08x", s.Channel, s.SourceFrame, got, s.Output)
		}
	}
}

func TestOriginalAVX512TransposeFullCapture(t *testing.T) {
	root := "../../private/cpu-numeric-ci-37927408109/cpu-original-replica-1/vocoder-trace"
	raw, err := os.ReadFile(root + "/operators.json")
	if os.IsNotExist(err) {
		t.Skip("private original CPU capture unavailable")
	}
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		ModelHash string `json:"model_sha256"`
		DLLHash   string `json:"dll_sha256"`
		Threads   int    `json:"original_threads"`
		Operators []struct {
			Node, Op string
			Names    []string
			Tensors  []nativeVocoderTensor
		}
	}
	if err := json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.ModelHash != "ff245a6149817c01c2d559e49087892623116766d1f3c95e7996bac23e6887e0" || ref.DLLHash != "00ca5c937940a94d052dbe52ec5fb0909db11ffbfccfd6e65e19b468169ff5e5" || ref.Threads != 1 {
		t.Fatal("original provenance differs")
	}
	g, err := OpenGraph("../../assets/synth/vocoder.onnx")
	if err != nil {
		t.Fatal(err)
	}
	h, count, total := sha256.New(), 0, 0
	for _, r := range ref.Operators {
		if r.Op != "ConvTranspose" {
			continue
		}
		count++
		t.Run(r.Node, func(t *testing.T) {
			x, want := readNativeVocoderTensor(t, root, r.Tensors[0]), readNativeVocoderTensor(t, root, r.Tensors[3])
			constants, err := g.Run(nil, r.Names[1:3])
			if err != nil {
				t.Fatal(err)
			}
			n := g.nodes[g.producer[r.Names[3]]]
			got, err := graphConvTranspose(n, []Value{x, constants[r.Names[1]], constants[r.Names[2]]}, true, 1, 1)
			if err != nil {
				t.Fatal(err)
			}
			if len(got[0].F) != len(want.F) {
				t.Fatal("original output extent differs")
			}
			var bits [4]byte
			for i, v := range want.F {
				b := math.Float32bits(v)
				binary.LittleEndian.PutUint32(bits[:], b)
				h.Write(bits[:])
				if math.Float32bits(got[0].F[i]) != b {
					t.Fatalf("output %d: got %08x want %08x", i, math.Float32bits(got[0].F[i]), b)
				}
			}
			total += len(want.F)
		})
	}
	if count != 10 || total != 1689600 || fmt.Sprintf("%x", h.Sum(nil)) != "e1a6f43bf0b068d634cab59d83f05a93c845b910d9df979a6e10c8d0cc3b5b7a" {
		t.Fatal("complete original output digest/count differs")
	}
}

func TestOriginalKernelProfileValidation(t *testing.T) {
	g := &Graph{}
	for _, p := range []OriginalKernelProfile{{Threads: 0}, {Threads: 2}, {Threads: 6, AVX512Convolutions: true}, {Threads: 6, FMAEmbeddings: true}, {Threads: 6, FMASmallConvolutions: true}, {Threads: 6, FMAProjections: true}} {
		if _, err := g.RunOriginalKernelsWithProfile(nil, []string{"x"}, p); err == nil {
			t.Fatal("unsupported profile was accepted")
		}
	}
}
