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

func TestDNNLGEMMNormalNOriginal(t *testing.T) {
	testDNNLGEMMOriginal(t, false)
}

func TestDNNLGEMMNormalTOriginal(t *testing.T) {
	testDNNLGEMMOriginal(t, true)
}

func testDNNLGEMMOriginal(t *testing.T, transposeB bool) {
	file, expectedHash := "testdata/dnnl_gemm_nn.json", "27ed683b7987411f0a5d91184c7453c22be348ccd47af3fd67d571ad7ca1e57b"
	if transposeB {
		file, expectedHash = "testdata/dnnl_gemm_nt.json", "6bad3a6782862a9f465a83f4736c17e5f57aedbb57f4caab49e6d521066b5171"
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var reference struct {
		DLLSHA256 string `json:"dll_sha256"`
		RVA       string
		TransB    string `json:"trans_b"`
		Cases     []struct {
			M, N, K, LDA, LDB, LDC int
			Seed                   uint32
			Beta                   float32
			Output                 []uint32
		}
	}
	if err := json.Unmarshal(data, &reference); err != nil {
		t.Fatal(err)
	}
	if reference.DLLSHA256 != "00ca5c937940a94d052dbe52ec5fb0909db11ffbfccfd6e65e19b468169ff5e5" || reference.RVA != "0xb36f4c0" || len(reference.Cases) != 496 {
		t.Fatal("unexpected original oneDNN reference provenance/count")
	}
	if transposeB && reference.TransB != "T" {
		t.Fatal("original oneDNN transposition differs")
	}
	hash, total := sha256.New(), 0
	for index, c := range reference.Cases {
		state := c.Seed
		fill := func(v []float32) {
			for i := range v {
				state ^= state << 13
				state ^= state >> 17
				state ^= state << 5
				v[i] = float32(int32(state)>>8) / 1048576
			}
		}
		bSize := (c.N-1)*c.LDB + c.K
		if transposeB {
			bSize = (c.K-1)*c.LDB + c.N
		}
		a, b, initial := make([]float32, (c.K-1)*c.LDA+c.M), make([]float32, bSize), make([]float32, (c.N-1)*c.LDC+c.M)
		fill(a)
		fill(b)
		fill(initial)
		kernel := DNNLGEMMNormalN
		if transposeB {
			kernel = DNNLGEMMNormalT
		}
		y, err := kernel(a, b, initial, c.M, c.N, c.K, c.LDA, c.LDB, c.LDC, c.Beta)
		if err != nil {
			t.Fatal(err)
		}
		if len(y) != len(c.Output) {
			t.Fatal("original oneDNN output count differs")
		}
		for i, v := range y {
			hash.Write(binary.LittleEndian.AppendUint32(nil, c.Output[i]))
			if bits := math.Float32bits(v); bits != c.Output[i] {
				t.Errorf("case %d m=%d n=%d k=%d output %d: got %08x want %08x", index, c.M, c.N, c.K, i, bits, c.Output[i])
				break
			}
		}
		total += len(y)
	}
	if total != 528966 || fmt.Sprintf("%x", hash.Sum(nil)) != expectedHash {
		t.Fatal("original oneDNN reference output count/hash differs")
	}
	t.Logf("Compared %d original oneDNN values; SHA256 %x", total, hash.Sum(nil))
}
