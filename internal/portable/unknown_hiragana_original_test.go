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

func TestUnknownHiraganaOriginal(t *testing.T) {
	raw, err := os.ReadFile("testdata/unknown_hiragana_original.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		DLL     string `json:"dll_sha256"`
		CPU     string
		Threads int  `json:"original_threads"`
		AVX512  bool `json:"avx512_convolutions"`
		Cases   []struct {
			Text, Voice    string
			Readings       []string
			Bytes          int
			SHA256         string
			WaveReport     string `json:"wave_report_sha256"`
			ReadingCapture string `json:"reading_capture_sha256"`
		}
	}
	if err = json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.DLL != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || ref.CPU != "AMD64 Family 25 Model 33 Stepping 0, AuthenticAMD" || ref.Threads != 6 || ref.AVX512 || len(ref.Cases) != 2 {
		t.Fatal("original capture profile differs")
	}
	e, err := Open(Options{AssetsDir: "../../assets", OriginalThreads: ref.Threads})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	for i, c := range ref.Cases {
		if c.Voice != []string{VoiceAkane, VoiceAoi}[i] || len(c.Readings) != 1 || c.Bytes < 44 || len(c.SHA256) != 64 || len(c.WaveReport) != 64 || len(c.ReadingCapture) != 64 {
			t.Fatal("original reference is incomplete")
		}
		t.Run(c.Voice, func(t *testing.T) {
			got, err := e.ReadingVoice(c.Text, c.Voice)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, c.Readings) {
				t.Fatalf("reading %q; want %q", got, c.Readings)
			}
			t.Run("WAV", func(t *testing.T) {
				if os.Getenv("AOIKANE_UNKNOWN_HIRAGANA_REFERENCE_AUDIT") != "1" {
					t.Skip("set AOIKANE_UNKNOWN_HIRAGANA_REFERENCE_AUDIT=1 for whole original WAV replay")
				}
				wav, err := e.Synthesize(context.Background(), engine.Request{Text: c.Text, Voice: c.Voice})
				if err != nil {
					t.Fatal(err)
				}
				if len(wav) != c.Bytes || fmt.Sprintf("%x", sha256.Sum256(wav)) != c.SHA256 {
					t.Fatalf("WAV differs: bytes=%d SHA256=%x", len(wav), sha256.Sum256(wav))
				}
				t.Logf("whole ordinary original WAV identical: %d bytes", len(wav))
			})
		})
	}
}
