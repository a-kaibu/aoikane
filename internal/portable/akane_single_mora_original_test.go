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

// These readings were captured from the original Akane frontend, including
// the random seed-2 failure; a single-mora fall retains the Kansai rise.
func TestAkaneSingleMoraReadingOriginal(t *testing.T) {
	raw, err := os.ReadFile("testdata/akane_single_mora_original.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		DLLHash  string `json:"dll_sha256"`
		Voice    string
		Captures []string `json:"capture_sha256"`
		Cases    []struct{ Text, Reading string }
	}
	if err = json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.DLLHash != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || ref.Voice != VoiceAkane || len(ref.Captures) != 2 || len(ref.Cases) != 4 {
		t.Fatal("original Akane single-mora reference provenance differs")
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

// Ordinary original direct-API output from the seed-2 differential corpus on
// AMD64 Family 25 Model 33, six threads; no instrumentation or injected tensors.
func TestAkaneSingleMoraOriginalWAV(t *testing.T) {
	if os.Getenv("AOIKANE_AKANE_RANDOM_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_AKANE_RANDOM_REFERENCE_AUDIT=1 for saved-original WAV replay")
	}
	e, err := Open(Options{AssetsDir: "../../assets", OriginalThreads: 6})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	got, err := e.Synthesize(context.Background(), engine.Request{Text: "のゆなむぷへぜつ。", Voice: VoiceAkane})
	if err != nil {
		t.Fatal(err)
	}
	const want = "bbd8fa59115d1b5dc2e42996261d93c37199c3bef4b9efd8d9ff4dd5aa49edfe"
	if len(got) != 215328 || fmt.Sprintf("%x", sha256.Sum256(got)) != want {
		t.Fatalf("original WAV differs: bytes=%d SHA256=%x", len(got), sha256.Sum256(got))
	}
	t.Logf("ordinary original full WAV identical: 215328 bytes SHA256=%s", want)
}
