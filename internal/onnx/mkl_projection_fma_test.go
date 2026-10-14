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

func TestOriginalProjectionFMAGraph(t *testing.T) {
	raw, err := os.ReadFile("testdata/mkl_projection_fma.json")
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
		Cases         []struct{ Operation, Rows, In, Out int }
	}
	if err = json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.DLLHash != "00ca5c937940a94d052dbe52ec5fb0909db11ffbfccfd6e65e19b468169ff5e5" || ref.ModelHash != "0341bdce841395eec505afab8d804038b681fe682cd1bb43894e63797bd4a142" || ref.WAVHash != "6be20e8718dc8a1329426baa824b9dabef10315dddc2099a3ad11ffac6032d7d" || ref.Threads != 1 || ref.Run != 14 || ref.Values != 13421 || len(ref.Cases) != 10 || ref.OutputHash != "b136ea4dae511633529a381d4a3be56e4c53a3a08ca7c14b9708f576fe39e862" {
		t.Fatal("ordinary original projection provenance differs")
	}
	compressed, err := os.ReadFile("testdata/mkl_projection_fma.bin.gz")
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
	if fmt.Sprintf("%x", sha256.Sum256(data)) != ref.ReferenceHash || ref.ReferenceHash != "0888afabdb422a96cc06c5ede2caba0ad2527b010f93931f7d0ae7f58f9d3c45" {
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
		if c.Operation <= last || c.Operation > 118 || (c.Rows != 11 && c.Rows != 199) || c.In < 1 || c.In > 256 || c.Out < 1 || c.Out > 256 {
			t.Fatal("original case geometry differs")
		}
		last = c.Operation
		input, weight, bias, want := take(c.In*c.Rows), take(c.Out*c.In), take(c.Out), take(c.Out*c.Rows)
		weightT := make([]float32, len(weight))
		for col := 0; col < c.Out; col++ {
			for k := 0; k < c.In; k++ {
				weightT[k*c.Out+col] = weight[col*c.In+k]
			}
		}
		g := &Graph{
			nodes:    []graphNode{{op: "MatMul", inputs: []string{"x", "w"}, outputs: []string{"product"}}, {op: "Add", inputs: []string{"product", "b"}, outputs: []string{"y"}}},
			initial:  map[string]Value{"w": {Kind: 1, Shape: []int64{int64(c.In), int64(c.Out)}, F: weightT}, "b": {Kind: 1, Shape: []int64{int64(c.Out)}, F: bias}},
			producer: map[string]int{"product": 0, "y": 1},
		}
		result, err := g.RunOriginalKernelsWithProfile(map[string]Value{"x": {Kind: 1, Shape: []int64{1, int64(c.Rows), int64(c.In)}, F: input}}, []string{"y"}, OriginalKernelProfile{Threads: 1, FMAProjections: true})
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
	t.Logf("Compared %d ordinary Intel original projection outputs; SHA256 %x", count, hash.Sum(nil))
}
