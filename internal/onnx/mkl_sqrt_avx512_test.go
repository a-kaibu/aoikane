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

func TestMKLSqrtAVX512Original(t *testing.T) {
	testOriginalAVX512Unary(t, "sqrt", "0x31300", 138210, 45, "dcc0ce2dfd01cc04528698490298967a7e6d6a21a1e0a371bea95bf0084c846c", MKLSqrtAVX512)
}

func TestMKLTanhAVX512Original(t *testing.T) {
	testOriginalAVX512Unary(t, "tanh", "0x33020", 139794, 45, "5bb96f3b255140ed4150542efa7083895c2090eac44c3610dc7a19c10b153f78", MKLTanhAVX512)
}

func testOriginalAVX512Unary(t *testing.T, name, rva string, count, caseCount int, expectedHash string, kernel func(float32) float32, graphOp ...string) {
	t.Helper()
	raw, err := os.ReadFile("testdata/mkl_" + name + "_avx512.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		DLLHash       string `json:"dll_sha256"`
		RVA           string `json:"entry_rva"`
		Mode          int
		Values        int
		Hash          string `json:"output_sha256"`
		ReferenceHash string `json:"reference_sha256"`
		Cases         []struct {
			Values    int
			Alignment int `json:"input_alignment"`
		}
	}
	if err = json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.DLLHash != "00ca5c937940a94d052dbe52ec5fb0909db11ffbfccfd6e65e19b468169ff5e5" || ref.RVA != rva || ref.Mode != 0x140102 || ref.Values != count || len(ref.Cases) != caseCount || ref.Hash != expectedHash {
		t.Fatal("original AVX512 unary provenance/count differs")
	}
	compressed, err := os.ReadFile("testdata/mkl_" + name + "_avx512.bin.gz")
	if err != nil {
		t.Fatal(err)
	}
	r, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(io.LimitReader(r, int64(ref.Values)*8+1))
	if err != nil {
		t.Fatal(err)
	}
	if err = r.Close(); err != nil {
		t.Fatal(err)
	}
	if len(data) != ref.Values*8 || fmt.Sprintf("%x", sha256.Sum256(data)) != ref.ReferenceHash {
		t.Fatal("original AVX512 unary reference storage differs")
	}
	hash, total := sha256.New(), 0
	var graphOutput []float32
	if len(graphOp) > 0 {
		input := make([]float32, ref.Values)
		for i := range input {
			input[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[i*8:]))
		}
		g := &Graph{nodes: []graphNode{{op: graphOp[0], inputs: []string{"x"}, outputs: []string{"y"}}}, initial: map[string]Value{}, producer: map[string]int{"y": 0}}
		out, err := g.RunOriginalKernelsWithProfile(map[string]Value{"x": {Kind: 1, Shape: []int64{int64(ref.Values)}, F: input}}, []string{"y"}, OriginalKernelProfile{Threads: 1, AVX512Unary: true})
		if err != nil {
			t.Fatal(err)
		}
		graphOutput = out["y"].F
		if len(graphOutput) != ref.Values {
			t.Fatal("graph output length differs")
		}
	}

	for ci, c := range ref.Cases {
		if c.Values < 1 || c.Values > ref.Values-total || c.Alignment < 0 || c.Alignment > 12 || c.Alignment%4 != 0 {
			t.Fatal("invalid original case")
		}
		for i := 0; i < c.Values; i++ {
			pair := data[total*8 : total*8+8]
			input, want := binary.LittleEndian.Uint32(pair), binary.LittleEndian.Uint32(pair[4:])
			got := math.Float32bits(kernel(math.Float32frombits(input)))
			if graphOutput != nil {
				got = math.Float32bits(graphOutput[total])
			}
			if got != want {
				t.Fatalf("case %d index %d input=%08x got=%08x want=%08x", ci, i, input, got, want)
			}
			hash.Write(pair[4:])
			total++
		}
	}
	if total != ref.Values || fmt.Sprintf("%x", hash.Sum(nil)) != ref.Hash {
		t.Fatal("original AVX512 unary count/hash differs")
	}
	t.Logf("Compared %d ordinary Intel original %s values; SHA256 %x", total, name, hash.Sum(nil))
}
