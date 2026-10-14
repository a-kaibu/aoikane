package dsp

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

// Compare every generated input and every exponent of the failing compressor
// invocation against the actual original imported function's binary32 result.
func TestMagnitudePowOriginal(t *testing.T) {
	meta, err := os.ReadFile("testdata/magnitude_pow_original.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		Count   int
		Base    int
		FMAFlag int    `json:"fma_flag"`
		SHA     string `json:"data_sha256"`
	}
	if err = json.Unmarshal(meta, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.Count != 65651 || ref.Base != 10 || ref.FMAFlag != 3 {
		t.Fatal("original powf profile changed")
	}
	compressed, err := os.ReadFile("testdata/magnitude_pow_original.bin.gz")
	if err != nil {
		t.Fatal(err)
	}
	z, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(z)
	if err != nil {
		t.Fatal(err)
	}
	if err = z.Close(); err != nil {
		t.Fatal(err)
	}
	if len(raw) != ref.Count*8 || fmt.Sprintf("%x", sha256.Sum256(raw)) != ref.SHA {
		t.Fatal("original reference is incomplete")
	}
	oldDifferences := 0
	for i := 0; i < ref.Count; i++ {
		x := math.Float32frombits(binary.LittleEndian.Uint32(raw[i*8:]))
		want := binary.LittleEndian.Uint32(raw[i*8+4:])
		got := math.Float32bits(originalMagnitudePow(x))
		if got != want {
			t.Errorf("case=%d exponent=%08x got=%08x original=%08x", i, math.Float32bits(x), got, want)
		}
		if math.Float32bits(float32(math.Pow(10, float64(x)))) != want {
			oldDifferences++
		}
	}
	if oldDifferences == 0 {
		t.Fatal("corpus does not expose old double-precision pow differences")
	}
	t.Logf("%d original binary32 results exact; old math.Pow differed on %d inputs", ref.Count, oldDifferences)
}
