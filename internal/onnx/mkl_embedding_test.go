package onnx

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestOriginalEmbeddingSynthetic(t *testing.T) {
	data, err := os.ReadFile("testdata/mkl_embedding.json")
	if err != nil {
		t.Fatal(err)
	}
	var reference struct {
		DLLSHA256 string `json:"dll_sha256"`
		RVA       string `json:"rva"`
		Threads   int    `json:"threads"`
		Cases     []struct {
			K, Rows int
			Seed    uint32 `json:"seed"`
			Hash    string `json:"output_sha256"`
			TransB  string `json:"trans_b"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &reference); err != nil {
		t.Fatal(err)
	}
	if reference.DLLSHA256 != "00ca5c937940a94d052dbe52ec5fb0909db11ffbfccfd6e65e19b468169ff5e5" || reference.RVA != "0x134700" || reference.Threads != 6 || len(reference.Cases) != 30 {
		t.Fatal("unexpected original embedding reference provenance/count")
	}
	for index, c := range reference.Cases {
		if c.Rows < 256 || c.Rows%6 > 1 || (c.K != 5 && c.K != 10 && c.K != 32) || (c.K == 32 && c.TransB != "T") || (c.K != 32 && c.TransB != "N") {
			t.Fatal("unexpected synthetic native projection geometry")
		}
		state := c.Seed
		fill := func(values []float32) {
			for i := range values {
				state ^= state << 13
				state ^= state >> 17
				state ^= state << 5
				values[i] = float32(int32(state)>>8) / 1048576
			}
		}
		a, input := make([]float32, 256*c.K), make([]float32, c.Rows*c.K)
		fill(a)
		fill(input)
		if c.TransB == "T" {
			logical := make([]float32, len(input))
			for row := 0; row < c.Rows; row++ {
				for k := 0; k < c.K; k++ {
					logical[row*c.K+k] = input[k*c.Rows+row]
				}
			}
			input = logical
		}
		l := &Linear{In: c.K, Out: 256, weight: make([]float32, len(a))}
		for col := 0; col < 256; col++ {
			for k := 0; k < c.K; k++ {
				l.weight[k*256+col] = a[col*c.K+k]
			}
		}
		y, err := l.RunOriginalEmbedding(input, c.Rows)
		if err != nil {
			t.Fatal(err)
		}
		hash := sha256.New()
		for _, value := range y {
			hash.Write(binary.LittleEndian.AppendUint32(nil, math.Float32bits(value)))
		}
		if got := fmt.Sprintf("%x", hash.Sum(nil)); got != c.Hash {
			t.Errorf("case %d k=%d rows=%d: got %s want %s", index, c.K, c.Rows, got, c.Hash)
		}
	}
}

// Real engine captures stay private. When available, this regression uses
// only Go and the captured reference; it never loads an original runtime.
func TestOriginalEmbeddingCaptured(t *testing.T) {
	for _, c := range []struct {
		trace  string
		values int
		hash   string
	}{
		{"torch-ops-feat-early", 7680, "bbc51c8bb39d4f5681d09084f086a74047cc3c30c93b453455b4c7909e6fd117"},
		{"torch-ops-feat-second", 9216, "65a46883031780e7f78c866be395cfff17524503e7d6b4799dbfbc8955bbc546"},
		{"torch-embedding-long", 41472, "f7d9a4f208cc844ba353d9d92d2d74147a105a7fc0753d5a1e35c3e44c4e5ae3"},
		{"torch-embedding-276", 211968, "e07e77209fa2137a6ac70525d44d171a0e20c63f8499eb9784ffe45a36dc35ce"},
		{"torch-embedding-266", 204288, "1e4dfadefecf3f69a00953d2754277e5c75b04539309bedba0307ec5ad487b0d"},
		{"torch-embedding-279", 214272, "504a4faaf47f7b6a2459ffc3e7d20a33a663e1033c2a0a1887e92fd496bedf1d"},
	} {
		t.Run(c.trace, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "..", "private", c.trace, "operations.json"))
			if os.IsNotExist(err) {
				t.Skip("private original engine capture unavailable")
			}
			if err != nil {
				t.Fatal(err)
			}
			var records []struct {
				Node       string    `json:"node"`
				InputShape []int64   `json:"input_shape"`
				Input      []float32 `json:"input"`
				Output     []float32 `json:"output"`
			}
			if err := json.Unmarshal(data, &records); err != nil {
				t.Fatal(err)
			}
			total, selected := 0, 0
			hash := sha256.New()
			for _, r := range records {
				if r.Node != "/model/emb_tok/art/block/MatMul" && r.Node != "/model/emb_tok/nima/block/MatMul" && r.Node != "/model/emb_tok/niab/block/MatMul" {
					continue
				}
				selected++
				if len(r.InputShape) != 3 || r.InputShape[0] != 1 {
					t.Fatal("invalid captured input shape")
				}
				l, err := OpenLinear(filepath.Join("..", "..", "assets", "synth", "featnet.onnx"), r.Node)
				if err != nil {
					t.Fatal(err)
				}
				y, err := l.RunOriginalEmbedding(r.Input, int(r.InputShape[1]))
				if err != nil {
					t.Fatal(err)
				}
				if len(y) != len(r.Output) {
					t.Fatal("captured output shape differs")
				}
				for i, value := range y {
					want := math.Float32bits(r.Output[i])
					if bits := math.Float32bits(value); bits != want {
						t.Fatalf("%s output %d: got %08x want %08x", r.Node, i, bits, want)
					}
					hash.Write(binary.LittleEndian.AppendUint32(nil, want))
				}
				total += len(y)
			}
			if total != c.values || selected != 3 || fmt.Sprintf("%x", hash.Sum(nil)) != c.hash {
				t.Fatal("original capture count/hash differs")
			}
		})
	}
}
