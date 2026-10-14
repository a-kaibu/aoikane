package onnx

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"testing"
)

func TestOriginalPointwiseFMAGraph(t *testing.T) {
	raw, err := os.ReadFile("testdata/conv_pointwise_fma.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		DLLHash       string `json:"dll_sha256"`
		ModelHash     string `json:"model_sha256"`
		WAVHash       string `json:"ordinary_wav_sha256"`
		Threads       int    `json:"original_threads"`
		Run, Values   int
		OutputHash    string `json:"output_sha256"`
		ReferenceHash string `json:"reference_sha256"`
		Cases         []struct{ Operation, Length, In, Out int }
	}
	if err = json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.DLLHash != "00ca5c937940a94d052dbe52ec5fb0909db11ffbfccfd6e65e19b468169ff5e5" || ref.ModelHash != "0341bdce841395eec505afab8d804038b681fe682cd1bb43894e63797bd4a142" || ref.WAVHash != "6be20e8718dc8a1329426baa824b9dabef10315dddc2099a3ad11ffac6032d7d" || ref.Threads != 1 || ref.Run != 14 || ref.Values != 231104 || len(ref.Cases) != 44 || ref.OutputHash != "90fbb217b8b70671eb4a84344be8b1204070016dbbf2ad06dd4d0d7a3cabba09" {
		t.Fatal("ordinary original pointwise provenance differs")
	}
	compressed, err := os.ReadFile("testdata/conv_pointwise_fma.bin.gz")
	if err != nil {
		t.Fatal(err)
	}
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(io.LimitReader(reader, 16<<20))
	if err != nil {
		t.Fatal(err)
	}
	if err = reader.Close(); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != ref.ReferenceHash || ref.ReferenceHash != "f3f1476debbfed31ee987eef5fa6b77104d430f54da5c8e7bcede35c82002e84" {
		t.Fatal("original reference storage differs")
	}
	take := func(n int) []float32 {
		if n < 0 || n > len(data)/4 {
			t.Fatal("truncated original reference")
		}
		v := make([]float32, n)
		for i := range v {
			v[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[i*4:]))
		}
		data = data[n*4:]
		return v
	}
	hash := sha256.New()
	count, last := 0, 0
	for _, c := range ref.Cases {
		if c.Operation <= last || c.Operation > 118 || (c.Length != 11 && c.Length != 199) || c.In < 1 || c.In > 256 || c.Out < 1 || c.Out > 256 {
			t.Fatal("original case geometry differs")
		}
		last = c.Operation
		input, weight, bias, want := take(c.In*c.Length), take(c.Out*c.In), take(c.Out), take(c.Out*c.Length)
		g := &Graph{
			nodes:    []graphNode{{op: "Conv", inputs: []string{"x", "w", "b"}, outputs: []string{"y"}}},
			initial:  map[string]Value{"w": {Kind: 1, Shape: []int64{int64(c.Out), int64(c.In), 1}, F: weight}, "b": {Kind: 1, Shape: []int64{int64(c.Out)}, F: bias}},
			producer: map[string]int{"y": 0},
		}
		result, err := g.RunOriginalKernelsWithProfile(map[string]Value{"x": {Kind: 1, Shape: []int64{1, int64(c.In), int64(c.Length)}, F: input}}, []string{"y"}, OriginalKernelProfile{Threads: 1, FMASmallConvolutions: true})
		if err != nil {
			t.Fatal(err)
		}
		got := result["y"].F
		if len(got) != len(want) {
			t.Fatal("original output length differs")
		}
		for j, v := range got {
			bits := math.Float32bits(v)
			if bits != math.Float32bits(want[j]) {
				t.Fatalf("operation %d index %d: got %08x want %08x", c.Operation, j, bits, math.Float32bits(want[j]))
			}
			hash.Write(binary.LittleEndian.AppendUint32(nil, bits))
			count++
		}
	}
	if len(data) != 0 || count != ref.Values || fmt.Sprintf("%x", hash.Sum(nil)) != ref.OutputHash {
		t.Fatal("original output count/hash differs")
	}
	t.Logf("Compared %d ordinary Intel original pointwise outputs; SHA256 %x", count, hash.Sum(nil))
}
