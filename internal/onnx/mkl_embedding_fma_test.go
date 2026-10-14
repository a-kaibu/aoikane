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

func TestOriginalEmbeddingFMA(t *testing.T) {
	raw, err := os.ReadFile("testdata/mkl_embedding_fma.json")
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
	if ref.DLLHash != "00ca5c937940a94d052dbe52ec5fb0909db11ffbfccfd6e65e19b468169ff5e5" || ref.ModelHash != "0341bdce841395eec505afab8d804038b681fe682cd1bb43894e63797bd4a142" || ref.WAVHash != "6be20e8718dc8a1329426baa824b9dabef10315dddc2099a3ad11ffac6032d7d" || ref.Threads != 1 || ref.Run != 14 || ref.Values != 8448 || len(ref.Cases) != 3 || ref.OutputHash != "6c8be4169f00311c685f01f46b194d316769ca16c7221fe1bd89a9c1d3f2490a" {
		t.Fatal("ordinary original FMA embedding provenance differs")
	}
	compressed, err := os.ReadFile("testdata/mkl_embedding_fma.bin.gz")
	if err != nil {
		t.Fatal(err)
	}
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(io.LimitReader(reader, 1<<20))
	if err != nil {
		t.Fatal(err)
	}
	if err = reader.Close(); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != ref.ReferenceHash || ref.ReferenceHash != "d5bdc78c1e9a039f12bfbf9845161278677956847fdf9f0a4dc3303a527edac6" {
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
	count := 0
	for i, c := range ref.Cases {
		if c.Operation != i+1 || c.Rows != 11 || c.Out != 256 || c.In != []int{32, 10, 5}[i] {
			t.Fatal("original case geometry differs")
		}
		input, originalWeight, want := take(c.Rows*c.In), take(c.Out*c.In), take(c.Rows*c.Out)
		l := &Linear{In: c.In, Out: c.Out, weight: make([]float32, c.In*c.Out)}
		for col := 0; col < c.Out; col++ {
			for k := 0; k < c.In; k++ {
				l.weight[k*c.Out+col] = originalWeight[col*c.In+k]
			}
		}
		g := &Graph{
			nodes:    []graphNode{{op: "MatMul", inputs: []string{"x", "w"}, outputs: []string{"y"}}},
			initial:  map[string]Value{"w": {Kind: 1, Shape: []int64{int64(c.In), int64(c.Out)}, F: l.weight}},
			producer: map[string]int{"y": 0},
		}
		result, err := g.RunOriginalKernelsWithProfile(map[string]Value{"x": {Kind: 1, Shape: []int64{1, int64(c.Rows), int64(c.In)}, F: input}}, []string{"y"}, OriginalKernelProfile{Threads: 1, FMAEmbeddings: true})
		if err != nil {
			t.Fatal(err)
		}
		got := result["y"].F
		if err != nil {
			t.Fatal(err)
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
	t.Logf("Compared %d ordinary Intel original embedding outputs; SHA256 %x", count, hash.Sum(nil))
}
