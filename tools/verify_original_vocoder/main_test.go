package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/a-kaibu/aoikane/internal/onnx"
)

func TestLocateChangedIntermediateAndRejectChangedReference(t *testing.T) {
	root := t.TempDir()
	var refs []tensor
	h := sha256.New()
	for i, values := range [][]float32{{1, 2}, {3, 4}, {5, 6}, {7, 8}, {9}} {
		data := make([]byte, len(values)*4)
		for j, v := range values {
			binary.LittleEndian.PutUint32(data[j*4:], math.Float32bits(v))
		}
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("model-%d.bin", i+1)), data, 0600); err != nil {
			t.Fatal(err)
		}
		refs = append(refs, tensor{ID: i + 1, DType: 6, Shape: []int64{int64(len(values))}, Stride: []int64{1}})
	}
	for _, id := range []int{1, 4, 2, 3} {
		b, _ := os.ReadFile(filepath.Join(root, fmt.Sprintf("model-%d.bin", id)))
		h.Write(b)
	}
	wave, _ := os.ReadFile(filepath.Join(root, "model-5.bin"))
	m := manifest{Threads: 1, Count: 8, Hash: fmt.Sprintf("%x", h.Sum(nil)), WaveHash: fmt.Sprintf("%x", sha256.Sum256(wave)), Wave: refs[4], Operators: []operator{{Operation: 10, Node: "conv", Op: "Conv", Names: []string{"x", "w", "b", "y"}, Tensors: refs[:4]}}}
	got := map[string]onnx.Value{}
	for i, name := range []string{"x", "w", "b", "y", "wave"} {
		v, err := load(root, refs[i])
		if err != nil {
			t.Fatal(err)
		}
		got[name] = v
	}
	r, err := compare(root, m, got)
	if err != nil || r.Different != 0 || r.First != nil {
		t.Fatalf("identical original: report=%+v error=%v", r, err)
	}
	v := got["y"]
	v.F[1] = math.Float32frombits(math.Float32bits(v.F[1]) + 1)
	got["y"] = v
	r, err = compare(root, m, got)
	if err != nil || r.Different != 1 || r.First == nil || r.First.Operation != 10 || r.First.Port != "output" || r.First.Difference.First != 1 {
		t.Fatalf("changed intermediate was not located: report=%+v error=%v", r, err)
	}
	wave[0] ^= 1
	if err := os.WriteFile(filepath.Join(root, "model-5.bin"), wave, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := compare(root, m, got); err == nil {
		t.Fatal("changed original wave was accepted")
	}
}

func TestOrdinaryOriginalVocoderManifest(t *testing.T) {
	root := "../../private/vocoder-observation-local"
	if _, err := os.Stat(filepath.Join(root, "operators.json")); os.IsNotExist(err) {
		t.Skip("private original capture unavailable")
	}
	args := []string{root, "../../assets/synth/vocoder.onnx", filepath.Join(root, "observation-validity.json"), filepath.Join(root, "ordinary-original.wav")}
	if _, err := readManifest(args[0], args[1], args[2], args[3], 1); err != nil {
		t.Fatal(err)
	}
	if _, err := readManifest(args[0], args[1], args[2], args[3], 6); err == nil {
		t.Fatal("wrong original thread profile was accepted")
	}
	invalid := filepath.Join(t.TempDir(), "invalid-observation.json")
	if err := os.WriteFile(invalid, []byte(`{"equal":false,"usable_as_numerical_reference":false}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readManifest(args[0], args[1], invalid, args[3], 1); err == nil {
		t.Fatal("invalid observation was accepted")
	}
}

func TestRejectSyntheticVocoderEvents(t *testing.T) {
	for _, e := range []string{"torch_vocoder_inputs_probe", "torch_vocoder_inputs_probe_restored", "torch_vocoder_f0_probe", "torch_vocoder_f0_probe_restored", "torch_control_probe_restored", "torch_compressor_probe", "torch_compressor_probe_restored", "torch_operation_calls", "forward_error"} {
		if ordinaryVocoderEvent(e, false) {
			t.Fatalf("nonordinary event %q accepted", e)
		}
	}
	if ordinaryVocoderEvent("future_probe", true) {
		t.Fatal("unknown synthetic event accepted")
	}
	if !ordinaryVocoderEvent("torch_operation_input", false) {
		t.Fatal("ordinary operator rejected")
	}
}
