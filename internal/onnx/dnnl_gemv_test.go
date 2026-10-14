package onnx

import (
	"encoding/json"
	"math"
	"os"
	"testing"
)

// Actual ordinary one-thread product output, including the bias added by
// oneDNN convolution. This fixture keeps the first mismatching tail channel
// and three other channels; no native library or private capture is needed.
func TestDNNLGEMVRowOriginalConvTail(t *testing.T) {
	raw, err := os.ReadFile("testdata/dnnl_gemv_tail.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		DLLSHA256                             string `json:"dll_sha256"`
		Node                                  string
		Frame, K                              int
		Channels, Input, Weight, Bias, Output []uint32
	}
	if err = json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.DLLSHA256 != "00ca5c937940a94d052dbe52ec5fb0909db11ffbfccfd6e65e19b468169ff5e5" || ref.Node != "/Conv_5" || ref.Frame != 504 || ref.K != 512 || len(ref.Channels) != 4 || len(ref.Output) != 4 || len(ref.Bias) != 4 {
		t.Fatal("unexpected original convolution tail provenance")
	}
	floats := func(bits []uint32) []float32 {
		values := make([]float32, len(bits))
		for i, b := range bits {
			values[i] = math.Float32frombits(b)
		}
		return values
	}
	got, err := dnnlGEMVRow(floats(ref.Input), floats(ref.Weight), nil, 4, ref.K, 1, ref.K, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	for i, value := range got {
		bits := math.Float32bits(float32(value + math.Float32frombits(ref.Bias[i])))
		if bits != ref.Output[i] {
			t.Fatalf("channel %d: Go=%08x original=%08x", ref.Channels[i], bits, ref.Output[i])
		}
	}
}
