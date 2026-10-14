package portable

import (
	"github.com/a-kaibu/aoikane/internal/engine"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"testing"
)

func TestSynthesisRejectsOriginalInvalidControls(t *testing.T) {
	e, err := Open(Options{AssetsDir: "../../assets"})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	for _, r := range []engine.Request{{Text: "あ。", Pitch: 0.01}, {Text: "あ。", Rate: 4.01}, {Text: "あ。", Volume: 2.01}, {Text: "あ。", Emphasis: 2.01}} {
		got, err := e.Synthesize(context.Background(), r)
		if err == nil || len(got) != 0 {
			t.Fatalf("original-invalid controls rendered audio: request=%+v err=%v bytes=%d", r, err, len(got))
		}
	}
}

// Native expectation was captured by TestRateBoundaryOriginalWAV using the
// ordinary local Akane SDK, then its complete WAV was compared byte for byte.
func TestRateBoundaryReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_CONTROLLED_REFERENCE_AUDIT") != "1" {
		t.Skip("strict saved original rate boundary WAV")
	}
	e, err := Open(Options{AssetsDir: "../../assets", OriginalThreads: 6})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	got, err := e.Synthesize(context.Background(), engine.Request{Text: "あ。", Voice: VoiceAkane, Rate: 4})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 85964 || fmt.Sprintf("%x", sha256.Sum256(got)) != "7fdb43556a3f230a955f2c4732accc22b5b27df84af51f6db7db3d9d31938047" {
		t.Fatalf("rate=4 original WAV differs: bytes=%d SHA256=%x", len(got), sha256.Sum256(got))
	}
}
