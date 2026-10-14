//go:build (windows || darwin || linux) && (amd64 || arm64)

package portable

import (
	"github.com/a-kaibu/aoikane/internal/difftest"
	"github.com/a-kaibu/aoikane/internal/engine"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// The ordinary original corpus is retained for strict cross-platform replay
// without the original DLL. It remains opt-in because eight syntheses are slow.
func TestAkaneRandomReferenceWAV(t *testing.T) {
	testAkaneRandomReferenceWAV(t, "testdata/akane_random_original.json", 1, 6)
}

func TestAkaneRandomSeed2ReferenceWAV(t *testing.T) {
	testAkaneRandomReferenceWAV(t, "testdata/akane_random_seed2_original.json", 2, 12)
}

func TestAkaneRandomSeed3ReferenceWAV(t *testing.T) {
	testAkaneRandomReferenceWAV(t, "testdata/akane_random_seed3_original.json", 3, 12)
}

func TestAkaneRandomSeed4ReferenceWAV(t *testing.T) {
	testAkaneRandomReferenceWAV(t, "testdata/akane_random_seed4_original.json", 4, 12)
}

func testAkaneRandomReferenceWAV(t *testing.T, file string, seed uint64, maxUnits int) {
	t.Helper()
	if os.Getenv("AOIKANE_AKANE_RANDOM_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_AKANE_RANDOM_REFERENCE_AUDIT=1 for the strict saved-original random audit")
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		CPU, Voice string
		DLLHash    string `json:"dll_sha256"`
		TorchHash  string `json:"torch_dll_sha256"`
		Threads    int    `json:"original_threads"`
		AVX512     bool   `json:"avx512_convolutions"`
		Seed       uint64
		Generator  int `json:"generator_version"`
		MaxUnits   int `json:"max_units"`
		Cases      []struct {
			Index  int
			Text   string
			Bytes  int
			SHA256 string
		}
	}
	if err = json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.DLLHash != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || ref.TorchHash != "00ca5c937940a94d052dbe52ec5fb0909db11ffbfccfd6e65e19b468169ff5e5" {
		t.Fatal("original DLL reference provenance differs")
	}
	if ref.CPU != "AMD64 Family 25 Model 33 Stepping 0, AuthenticAMD" || ref.Voice != VoiceAkane || ref.Threads != 6 || ref.AVX512 || ref.Seed != seed || ref.Generator != difftest.GeneratorVersion || ref.MaxUnits != maxUnits || len(ref.Cases) != 8 {
		t.Fatal("original Akane random corpus profile differs")
	}
	texts := difftest.Texts(ref.Seed, 8, ref.MaxUnits)
	e, err := Open(Options{AssetsDir: "../../assets", OriginalThreads: ref.Threads})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	for i, c := range ref.Cases {
		if c.Index != i || c.Text != texts[i] || c.Bytes < 44 || len(c.SHA256) != 64 {
			t.Fatal("original random corpus differs from the seeded generator")
		}
		t.Run(fmt.Sprintf("case-%04d", c.Index), func(t *testing.T) {
			got, err := e.Synthesize(context.Background(), engine.Request{Text: c.Text, Voice: ref.Voice})
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != c.Bytes || fmt.Sprintf("%x", sha256.Sum256(got)) != c.SHA256 {
				t.Fatalf("%q differs: bytes=%d SHA256=%x; want bytes=%d SHA256=%s", c.Text, len(got), sha256.Sum256(got), c.Bytes, c.SHA256)
			}
			t.Logf("ordinary original Akane full WAV identical: %q (%d bytes)", c.Text, len(got))
		})
	}
}
