package onnx

import (
	"encoding/json"
	"math"
	"os"
	"testing"
)

func TestMKLGEMMTransposeNOriginal(t *testing.T) {
	testOriginalGEMM(t, "testdata/mkl_gemm_tn.json", false)
}

func TestMKLGEMMTransposeTOriginal(t *testing.T) {
	testOriginalGEMM(t, "testdata/mkl_gemm_tt.json", true)
}

func testOriginalGEMM(t *testing.T, path string, transposeB bool) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var reference struct {
		DLLSHA256 string `json:"dll_sha256"`
		RVA       string `json:"rva"`
		TransB    string `json:"trans_b"`
		Cases     []struct {
			M, N, K, LDA, LDB int
			BAlignment        int      `json:"b_alignment"`
			Seed              uint32   `json:"seed"`
			Alpha             float32  `json:"alpha"`
			Output            []uint32 `json:"output"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &reference); err != nil {
		t.Fatal(err)
	}
	if reference.DLLSHA256 != "00ca5c937940a94d052dbe52ec5fb0909db11ffbfccfd6e65e19b468169ff5e5" || reference.RVA != "0x2824320" || len(reference.Cases) != 352 {
		t.Fatal("unexpected original GEMM reference provenance/count")
	}
	if (transposeB && reference.TransB != "T") || (!transposeB && reference.TransB != "" && reference.TransB != "N") {
		t.Fatal("unexpected original GEMM transpose reference")
	}
	for index, c := range reference.Cases {
		state := c.Seed
		fill := func(values []float32) {
			for i := range values {
				state ^= state << 13
				state ^= state >> 17
				state ^= state << 5
				values[i] = float32(int32(state)>>8) / 1048576
			}
		}
		a := make([]float32, (c.M-1)*c.LDA+c.K)
		span := (c.N-1)*c.LDB + c.K
		if transposeB {
			span = (c.K-1)*c.LDB + c.N
		}
		b := make([]float32, span)
		fill(a)
		fill(b)
		var y []float32
		if transposeB {
			y, err = MKLGEMMTransposeT(a, b, c.M, c.N, c.K, c.LDA, c.LDB, c.Alpha)
		} else {
			y, err = MKLGEMMTransposeN(a, b, c.M, c.N, c.K, c.LDA, c.LDB, c.BAlignment, c.Alpha)
		}
		if err != nil {
			t.Fatalf("case %d: %v", index, err)
		}
		if len(c.Output) != len(y) {
			t.Fatalf("case %d: unexpected output count", index)
		}
		for j, value := range y {
			if bits := math.Float32bits(value); bits != c.Output[j] {
				t.Errorf("case %d m=%d n=%d k=%d lda=%d ldb=%d align=%d output %d: got %08x want %08x", index, c.M, c.N, c.K, c.LDA, c.LDB, c.BAlignment, j, bits, c.Output[j])
				break
			}
		}
	}
}
