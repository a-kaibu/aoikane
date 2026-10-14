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

func TestLeadingZeroReadingOriginal(t *testing.T) {
	raw, err := os.ReadFile("testdata/leading_zero_reading_original.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		DLLHash  string   `json:"dll_sha256"`
		Captures []string `json:"capture_sha256"`
		Cases    []struct{ Text, Voice, Reading string }
	}
	if err = json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.DLLHash != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(ref.Captures) != 2 || len(ref.Cases) != 10 {
		t.Fatal("original leading-zero reference provenance differs")
	}
	e, err := Open(Options{AssetsDir: "../../assets"})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	for _, c := range ref.Cases {
		got, err := e.ReadingVoice(c.Text, c.Voice)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0] != c.Reading {
			t.Errorf("%s %q: got %q want %q", c.Voice, c.Text, got, c.Reading)
		}
	}
}

// Saved ordinary original API output: Akane random seed 2, case 6, AMD64
// Family 25 Model 33 with six native threads, without instrumentation.
func TestLeadingZeroOriginalWAV(t *testing.T) {
	if os.Getenv("AOIKANE_AKANE_RANDOM_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_AKANE_RANDOM_REFERENCE_AUDIT=1 for saved-original WAV replay")
	}
	e, err := Open(Options{AssetsDir: "../../assets", OriginalThreads: 6})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	got, err := e.Synthesize(context.Background(), engine.Request{Text: "！（A０0。", Voice: VoiceAkane})
	if err != nil {
		t.Fatal(err)
	}
	const want = "d2cacebf15d4c3688e178933a82e5d9d278f1b628994a0b13ce57ae823e4f7ae"
	if len(got) != 254542 || fmt.Sprintf("%x", sha256.Sum256(got)) != want {
		t.Fatalf("original WAV differs: bytes=%d SHA256=%x", len(got), sha256.Sum256(got))
	}
	t.Logf("ordinary original full WAV identical: 254542 bytes SHA256=%s", want)
}
