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

func TestMKLGEMMNormalTOriginal(t *testing.T)   { testOriginalNormalTGEMM(t, false) }
func TestMKLSmallKNormalTOriginal(t *testing.T) { testOriginalNormalTGEMM(t, true) }
func testOriginalNormalTGEMM(t *testing.T, worker bool) {
	t.Helper()
	path, rva, count := "testdata/mkl_gemm_nt.json", "0x2824320", 352
	if worker {
		path, rva, count = "testdata/mkl_smallk_nt.json", "0x789270", 448
	}
	expectedValues, expectedHash := 37620, "6465ab1a3cb834cae7d6969288ac55fb7db7eb810e5fd3bd1abfa4c685c18d47"
	if worker {
		expectedValues, expectedHash = 126720, "8ac6591b0b41cb85c66ec7e230c92e8149d87c9ba622bf3c3dea9e8f1f6faecc"
	}
	h, total := sha256.New(), 0
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
	if reference.DLLSHA256 != "00ca5c937940a94d052dbe52ec5fb0909db11ffbfccfd6e65e19b468169ff5e5" || reference.RVA != rva || reference.TransA != "N" || reference.TransB != "T" || len(reference.Cases) != count {
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
		a, b, initial := make([]float32, (c.K-1)*c.LDA+c.M), make([]float32, (c.K-1)*c.LDB+c.N), make([]float32, (c.N-1)*c.LDC+c.M)
		fill(a)
		fill(b)
		fill(initial)
		y, err := MKLGEMMNormalT(a, b, initial, c.M, c.N, c.K, c.LDA, c.LDB, c.LDC, c.CAlignment, c.Alpha, c.Beta)
		if worker {
			if c.Alpha != 1 || (c.Beta != 0 && c.Beta != 1) {
				t.Fatal("unexpected original worker scales")
			}
			y, err = MKLSmallKNormalT(a, b, initial, c.M, c.N, c.K, c.LDA, c.LDB, c.LDC, c.CAlignment, c.Beta)
		}
		if err != nil {
			t.Fatalf("case %d: %v", index, err)
		}
		if len(y) != len(c.Output) {
			t.Fatalf("case %d: output count differs", index)
		}
		for i, value := range y {
			h.Write(binary.LittleEndian.AppendUint32(nil, c.Output[i]))
			total++
			if bits := math.Float32bits(value); bits != c.Output[i] {
				t.Errorf("case %d m=%d n=%d k=%d ldc=%d alignment=%d alpha=%g beta=%g output %d: got %08x want %08x", index, c.M, c.N, c.K, c.LDC, c.CAlignment, c.Alpha, c.Beta, i, bits, c.Output[i])
				break
			}
		}
	}
	if total != expectedValues || fmt.Sprintf("%x", h.Sum(nil)) != expectedHash {
		t.Fatal("original N,T output count/hash differs")
	}
	t.Logf("Compared %d original MKL values; SHA256 %x", total, h.Sum(nil))

}
