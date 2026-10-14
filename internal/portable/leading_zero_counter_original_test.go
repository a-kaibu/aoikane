package portable

import (
	"github.com/a-kaibu/aoikane/internal/engine"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func TestLeadingZeroCounterReadingOriginal(t *testing.T) {
	testLeadingZeroReadingOriginal(t, "testdata/leading_zero_counter_original.json", 20, 2)
}

func TestKanjiLeadingZeroReadingOriginal(t *testing.T) {
	testLeadingZeroReadingOriginal(t, "testdata/kanji_leading_zero_original.json", 32, 3)
}

func testLeadingZeroReadingOriginal(t *testing.T, file string, count, captures int) {
	t.Helper()
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		DLLHash   string `json:"dll_sha256"`
		Voice     string
		Captures  []string `json:"capture_sha256"`
		RulesHash string   `json:"rules_sha256"`
		Cases     []struct{ Text, Reading string }
	}
	if err = json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.DLLHash != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || ref.Voice != VoiceAkane || len(ref.Captures) != captures || len(ref.Cases) != count {
		t.Fatal("original leading-zero counter reference provenance differs")
	}
	rules, err := os.ReadFile("../../assets/rules/bancharcx-kansai.txt")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(rules)) != ref.RulesHash {
		t.Fatal("original Kansai numeral rules differ")
	}
	e, err := Open(Options{AssetsDir: "../../assets"})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	for _, c := range ref.Cases {
		got, err := e.ReadingVoice(c.Text, ref.Voice)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0] != c.Reading {
			t.Errorf("%q: got %q want %q", c.Text, got, c.Reading)
		}
	}
}

func TestLeadingZeroCounterOriginalWAV(t *testing.T) {
	testLeadingZeroOriginalWAV(t, "testdata/leading_zero_counter_wav_original.json", 3)
}

func TestKanjiLeadingZeroOriginalWAV(t *testing.T) {
	testLeadingZeroOriginalWAV(t, "testdata/kanji_leading_zero_wav_original.json", 7)
}

func testLeadingZeroOriginalWAV(t *testing.T, file string, count int) {
	t.Helper()
	if os.Getenv("AOIKANE_AKANE_RANDOM_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_AKANE_RANDOM_REFERENCE_AUDIT=1 for saved-original WAV replay")
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
		Cases      []struct {
			Text   string
			Bytes  int
			SHA256 string
		}
	}
	if err = json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.CPU != "AMD64 Family 25 Model 33 Stepping 0, AuthenticAMD" || ref.Voice != VoiceAkane || ref.Threads != 6 || ref.AVX512 || len(ref.Cases) != count || ref.DLLHash != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || ref.TorchHash != "00ca5c937940a94d052dbe52ec5fb0909db11ffbfccfd6e65e19b468169ff5e5" {
		t.Fatal("ordinary original leading-zero counter WAV provenance differs")
	}
	e, err := Open(Options{AssetsDir: "../../assets", OriginalThreads: ref.Threads})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	for _, c := range ref.Cases {
		t.Run(c.Text, func(t *testing.T) {
			got, err := e.Synthesize(context.Background(), engine.Request{Text: c.Text, Voice: ref.Voice})
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != c.Bytes || fmt.Sprintf("%x", sha256.Sum256(got)) != c.SHA256 {
				t.Fatalf("original WAV differs: bytes=%d SHA256=%x", len(got), sha256.Sum256(got))
			}
			t.Logf("ordinary original full WAV identical: %d bytes SHA256=%s", len(got), c.SHA256)
		})
	}
}
