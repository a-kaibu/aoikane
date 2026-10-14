package onnx

import (
	"encoding/json"
	"math"
	"os"
	"testing"
)

func TestMKLGEMMNormalNOriginal(t *testing.T) {
	testOriginalNormalGEMM(t, false)
}

func TestMKLSmallKNormalNOriginal(t *testing.T) {
	testOriginalNormalGEMM(t, true)
}

func testOriginalNormalGEMM(t *testing.T, worker bool) {
	t.Helper()
	path, rva, count := "testdata/mkl_gemm_nn.json", "0x2824320", 352
	if worker {
		path, rva, count = "testdata/mkl_smallk_nn.json", "0x789270", 448
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var reference struct {
		DLLSHA256 string `json:"dll_sha256"`
		RVA       string `json:"rva"`
		TransA    string `json:"trans_a"`
		TransB    string `json:"trans_b"`
		Cases     []struct {
			M, N, K, LDA, LDB, LDC int
			CAlignment             int    `json:"c_alignment"`
			Seed                   uint32 `json:"seed"`
			Alpha, Beta            float32
			Output                 []uint32
		}
	}
	if err := json.Unmarshal(data, &reference); err != nil {
		t.Fatal(err)
	}
	if reference.DLLSHA256 != "00ca5c937940a94d052dbe52ec5fb0909db11ffbfccfd6e65e19b468169ff5e5" || reference.RVA != rva || reference.TransA != "N" || reference.TransB != "N" || len(reference.Cases) != count {
		t.Fatal("unexpected original normal GEMM provenance/count")
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
		a, b, initial := make([]float32, (c.K-1)*c.LDA+c.M), make([]float32, (c.N-1)*c.LDB+c.K), make([]float32, (c.N-1)*c.LDC+c.M)
		fill(a)
		fill(b)
		fill(initial)
		y, err := MKLGEMMNormalN(a, b, initial, c.M, c.N, c.K, c.LDA, c.LDB, c.LDC, c.CAlignment, c.Alpha, c.Beta)
		if worker {
			if c.Alpha != 1 || c.Beta != 1 {
				t.Fatal("unexpected original worker scales")
			}
			y, err = MKLSmallKNormalN(a, b, initial, c.M, c.N, c.K, c.LDA, c.LDB, c.LDC, c.CAlignment)
		}
		if err != nil {
			t.Fatalf("case %d: %v", index, err)
		}
		if len(y) != len(c.Output) {
			t.Fatalf("case %d: output count differs", index)
		}
		for i, value := range y {
			if bits := math.Float32bits(value); bits != c.Output[i] {
				t.Errorf("case %d m=%d n=%d k=%d ldc=%d alignment=%d alpha=%g beta=%g output %d: got %08x want %08x", index, c.M, c.N, c.K, c.LDC, c.CAlignment, c.Alpha, c.Beta, i, bits, c.Output[i])
				break
			}
		}
	}
}
