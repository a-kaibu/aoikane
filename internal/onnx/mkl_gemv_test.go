package onnx

import (
	"encoding/json"
	"math"
	"os"
	"testing"
)

// The checked-in reference contains only synthetic inputs generated from a
// public xorshift seed, and original DLL output bits. No model assets are used.
func TestMKLGEMVTransposeOriginal(t *testing.T) {
	testOriginalGEMV(t, "testdata/mkl_gemv.json", true)
}

func TestMKLGEMVNormalOriginal(t *testing.T) {
	testOriginalGEMV(t, "testdata/mkl_gemv_normal.json", false)
}

func testOriginalGEMV(t *testing.T, path string, transpose bool) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var reference struct {
		DLLSHA256 string `json:"dll_sha256"`
		RVA       string `json:"rva"`
		Trans     string `json:"trans"`
		Cases     []struct {
			M, N, LDA, IncX int
			AAlignment      int      `json:"a_alignment"`
			XAlignment      int      `json:"x_alignment"`
			YAlignment      int      `json:"y_alignment"`
			Seed            uint32   `json:"seed"`
			Alpha           float32  `json:"alpha"`
			Beta            float32  `json:"beta"`
			Output          []uint32 `json:"output"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &reference); err != nil {
		t.Fatal(err)
	}
	count := 528
	if !transpose {
		count = 792
	}
	if reference.DLLSHA256 != "00ca5c937940a94d052dbe52ec5fb0909db11ffbfccfd6e65e19b468169ff5e5" || reference.RVA != "0x86f060" || len(reference.Cases) != count {
		t.Fatal("unexpected original GEMV reference provenance/count")
	}
	if (!transpose && reference.Trans != "N") || (transpose && reference.Trans != "" && reference.Trans != "T") {
		t.Fatal("unexpected original GEMV transpose reference")
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
		a := make([]float32, (c.N-1)*c.LDA+c.M)
		xCount := c.M
		if !transpose {
			xCount = c.N
		}
		x := make([]float32, (xCount-1)*c.IncX+1)
		fill(a)
		fill(x)
		var y []float32
		if transpose {
			y, err = MKLGEMVTranspose(a, x, c.M, c.N, c.LDA, c.IncX, c.AAlignment, c.XAlignment, c.Alpha)
		} else {
			if c.IncX != 1 {
				t.Fatal("unsupported normal GEMV input stride")
			}
			initial := make([]float32, c.M)
			fill(initial)
			y, err = MKLGEMVNormal(a, x, initial, c.M, c.N, c.LDA, c.YAlignment, c.Alpha, c.Beta)
		}
		if err != nil {
			t.Fatalf("case %d: %v", index, err)
		}
		if len(c.Output) != len(y) {
			t.Fatalf("case %d: unexpected output count", index)
		}
		for j, value := range y {
			if bits := math.Float32bits(value); bits != c.Output[j] {
				t.Errorf("case %d m=%d n=%d lda=%d incx=%d align=%d/%d output %d: got %08x want %08x", index, c.M, c.N, c.LDA, c.IncX, c.AAlignment, c.XAlignment, j, bits, c.Output[j])
				break
			}
		}
	}
}
