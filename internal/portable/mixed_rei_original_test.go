package portable

import (
	"github.com/a-kaibu/aoikane/internal/engine"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"testing"
)

func TestMixedReiReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_MIXED_REI_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_MIXED_REI_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/mixed_rei_wave_original.json", 6)
}

func TestMixedReiUnitCounterReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_MIXED_REI_UNIT_COUNTER_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_MIXED_REI_UNIT_COUNTER_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/mixed_rei_unit_counter_wave_original.json", 8)
}

func TestMixedReiUnitBranchesReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_MIXED_REI_UNIT_BRANCHES_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_MIXED_REI_UNIT_BRANCHES_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/mixed_rei_unit_branches_wave_original.json", 8)
}

func testMixedReiReferenceWAV(t *testing.T, path string, count int) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		CPU     string
		DLL     string   `json:"dll_sha256"`
		Reports []string `json:"report_sha256"`
		Threads int      `json:"original_threads"`
		AVX512  bool     `json:"avx512_convolutions"`
		Cases   []struct {
			Text, Voice string
			Bytes       int
			SHA256      string
		}
	}
	if err = json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.DLL != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(ref.Reports) != 2 || ref.CPU != "AMD64 Family 25 Model 33 Stepping 0, AuthenticAMD" || ref.Threads != 6 || ref.AVX512 || len(ref.Cases) != count {
		t.Fatal("ordinary original WAV profile differs")
	}
	e, err := Open(Options{AssetsDir: "../../assets", OriginalThreads: ref.Threads})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	for i, c := range ref.Cases {
		if (c.Voice != VoiceAkane && c.Voice != VoiceAoi) || c.Bytes < 44 || len(c.SHA256) != 64 {
			t.Fatal("original WAV reference is incomplete")
		}
		t.Run(fmt.Sprintf("%s/case-%d", c.Voice, i), func(t *testing.T) {
			got, err := e.Synthesize(context.Background(), engine.Request{Text: c.Text, Voice: c.Voice})
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != c.Bytes || fmt.Sprintf("%x", sha256.Sum256(got)) != c.SHA256 {
				t.Fatalf("whole original WAV differs: bytes=%d SHA256=%x; want %d %s", len(got), sha256.Sum256(got), c.Bytes, c.SHA256)
			}
			t.Logf("whole original WAV identical: %q, %d bytes", c.Text, len(got))
		})
	}
}

// All 48 native captures stay in the regular regression, including small-unit
// and leading-zero counter branches. No case is filtered by result.
func TestMixedReiReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "mixed-rei-seed15", 48, "712f88faf73ef06b6c7a0918014b16f54ad4075cf0d6cb3688cc46a0f3fbda88")
}

func TestMixedReiUnitReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "mixed-rei-unit-64", 64, "d5064eab0145aafeba64c2044121c31fc9645fa8089be677203f38675979d2ed")
}

func testMixedReiReadingOriginal(t *testing.T, corpusName string, count int, corpusSHA string) {
	t.Helper()
	e, err := Open(Options{AssetsDir: "../../assets"})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	for _, voice := range []string{"akane", "aoi"} {
		raw, err := os.ReadFile(fmt.Sprintf("../../tools/compare_reading/testdata/%s-%s-original.json", corpusName, voice))
		if err != nil {
			t.Fatal(err)
		}
		var ref struct {
			Voice    string
			DLL      string `json:"dll_sha256"`
			Corpus   string `json:"corpus_sha256"`
			Complete bool
			Cases    []struct {
				Index    int
				Text     string
				Readings []string
			}
		}
		if err := json.Unmarshal(raw, &ref); err != nil {
			t.Fatal(err)
		}
		wantVoice := VoiceAkane
		if voice == "aoi" {
			wantVoice = VoiceAoi
		}
		if !ref.Complete || ref.Voice != wantVoice || ref.DLL != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || ref.Corpus != corpusSHA || len(ref.Cases) != count {
			t.Fatal("original mixed 零 capture is incomplete")
		}
		for i, c := range ref.Cases {
			if c.Index != i || c.Text == "" || len(c.Readings) == 0 {
				t.Fatal("original mixed 零 ordering differs")
			}
			got, err := e.ReadingVoice(c.Text, ref.Voice)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, c.Readings) {
				t.Errorf("%s case %d %q: got %q; want %q", ref.Voice, i, c.Text, got, c.Readings)
			}
		}
	}
}
