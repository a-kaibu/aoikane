package synthesis

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/a-kaibu/aoikane/internal/onnx"
)

type capturedTensor struct {
	ID, DType     int
	Shape, Stride []int64
	Tokens        map[string]capturedTensor
	Members       []capturedTensor
}

// Decode the logical tensor, including original transposed output strides.
func capturedBytes(t *testing.T, root string, tensor capturedTensor) []byte {
	t.Helper()
	size := 4
	if tensor.DType == 4 {
		size = 8
	} else if tensor.DType != 6 {
		t.Fatal("unsupported captured dtype")
	}
	if len(tensor.Shape) != len(tensor.Stride) {
		t.Fatal("invalid captured stride")
	}
	raw, err := os.ReadFile(filepath.Join(root, fmt.Sprintf("model-%d.bin", tensor.ID)))
	if err != nil {
		t.Fatal(err)
	}
	count := int64(1)
	for _, n := range tensor.Shape {
		if n < 1 || n > (32<<20)/count {
			t.Fatal("invalid captured shape")
		}
		count *= n
	}
	out := make([]byte, int(count)*size)
	for i := int64(0); i < count; i++ {
		remain, offset := i, int64(0)
		for j := len(tensor.Shape) - 1; j >= 0; j-- {
			offset += remain % tensor.Shape[j] * tensor.Stride[j]
			remain /= tensor.Shape[j]
		}
		if offset < 0 || offset >= int64(len(raw)/size) {
			t.Fatal("truncated original tensor")
		}
		copy(out[int(i)*size:], raw[int(offset)*size:int(offset+1)*size])
	}
	return out
}

// Exercise the production tensor adapter and kernel selection against all
// seven original TorchScript outputs. No vocoder or runtime DLL is opened.
func TestAcousticFeaturesOriginal(t *testing.T) {
	for _, tc := range []struct {
		name, hash string
		count      int
	}{
		{"torch-unary-structured", "4119ab073970425b2ec0958cf0f2e3f64c86381c34f67f67a152294a2edf6b4c", 10415},
		{"compressor-operators-short", "3d448f7df62618cc05b36ca9d39813edf0bdbbfa07861b2949c91bf6c9ec8473", 5034},
		{"featnet-four-phones", "afd5953d7a1782ac5ed012cc7fa81dda85c0848cb5f6bdf36725d866774a7acb", 6846},
		{"featnet-seven-phones", "a0cf56369578e108a3713b27ae78493c80defc73010ca5072c5f431ef2a7e42b", 14560},
		{"featnet-eight-phones", "1ebd8477cec0c2c89d57e5b8355cabd4e69dccfb2e59f808bb16f155f8f1df64", 16238},
		{"torch-controls-f0", "e407d1c93f424d1dfd9a9cba1c6bf82491adb5bd270ad2b91b05785de35c00cb", 10415},
		{"torch-controls-mixed", "792da8f8ffbb9f5a91354fcf793930fe1050f79315e8a37c48570d63fe76b798", 11085},
		{"torch-controls-overrides", "26d0e5e156add4ec847dec94b0c09e0ecacdd32ca1e1c4d9f37297c95361635a", 7333},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := filepath.Join("..", "..", "private", tc.name)
			data, err := os.ReadFile(filepath.Join(root, "events.json"))
			if os.IsNotExist(err) {
				t.Skip("private original capture unavailable")
			}
			if err != nil {
				t.Fatal(err)
			}
			var events []json.RawMessage
			if err := json.Unmarshal(data, &events); err != nil {
				t.Fatal(err)
			}
			var inputs []capturedTensor
			var result capturedTensor
			run := 0
			for _, raw := range events {
				var header struct{ Event string }
				if err := json.Unmarshal(raw, &header); err != nil {
					t.Fatal(err)
				}
				if header.Event != "torch_jit_forward" && header.Event != "torch_jit_result" {
					continue
				}
				var e struct {
					Event  string
					Run    int
					Inputs []capturedTensor
					Output capturedTensor
				}
				if err := json.Unmarshal(raw, &e); err != nil {
					t.Fatal(err)
				}
				if e.Event == "torch_jit_forward" && len(e.Inputs) == 10 {
					if run != 0 {
						t.Fatal("expected one original FeatNet call")
					}
					run, inputs = e.Run, e.Inputs
				}
				if e.Event == "torch_jit_result" && run != 0 && e.Run == run {
					result = e.Output
				}
			}
			if len(inputs) != 10 || len(result.Members) != 7 {
				t.Fatal("missing original FeatNet call")
			}
			named := inputs[0].Tokens
			for i, name := range []string{"sty", "speed", "pitch", "into", "loud", "dur", "f0c", "pwr"} {
				named[name] = inputs[i+2]
			}
			var tensors []modelTensor
			for name, original := range named {
				kind := 1
				if original.DType == 4 {
					kind = 7
				}
				tensors = append(tensors, modelTensor{Name: name, Type: kind, Shape: original.Shape, Data: capturedBytes(t, root, original)})
			}
			graph, err := onnx.OpenGraph("../../assets/synth/featnet.onnx")
			if err != nil {
				t.Fatal(err)
			}
			engine := Engine{featnet: graph}
			got, err := engine.acousticFeatures(tensors)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 7 {
				t.Fatal("invalid production output count")
			}
			hash, total := sha256.New(), 0
			for i, output := range got {
				original := result.Members[i]
				want := capturedBytes(t, root, original)
				if original.DType != 6 || !slices.Equal(original.Shape, output.Shape) || len(want) != len(output.Data)*4 {
					t.Fatalf("%s shape/type differs", output.Name)
				}
				for j, x := range output.Data {
					bits := binary.LittleEndian.Uint32(want[j*4:])
					if math.Float32bits(x) != bits {
						t.Fatalf("%s[%d] got %08x want %08x", output.Name, j, math.Float32bits(x), bits)
					}
				}
				hash.Write(want)
				total += len(output.Data)
			}
			if total != tc.count || fmt.Sprintf("%x", hash.Sum(nil)) != tc.hash {
				t.Fatal("original output count/hash differs")
			}
		})
	}
}
