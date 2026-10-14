//go:build (windows || darwin || linux) && (amd64 || arm64)

package portable

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/a-kaibu/aoikane/internal/engine"
)

// References come from fresh, uninstrumented original CLI invocations with
// these public settings, rather than synthetic acoustic-model control inputs.
func TestOriginalWAVControls(t *testing.T) {
	var e *Engine
	t.Cleanup(func() {
		if e != nil {
			e.Close()
		}
	})
	for _, tc := range []struct {
		name, hash                    string
		samples                       int
		rate, pitch, volume, emphasis float64
		text                          string
	}{
		{"rate", "cfb065dab20e2e0f3d05157fa9d9107b26cc6ab5236fbf5985b223468f14a980", 46320, 1.5, 1, 1, 1, "あ。"},
		{"pitch", "890e4bba0e9816749338e661da0613125c7819f3f2c7119a9fe14deda8669c85", 49200, 1, 1.2, 1, 1, "あ。"},
		{"volume", "a4a7803fb7d8d5810d6b87f09b6bf5e98df0b5174b504cf3b551b2fd2b839ae0", 49200, 1, 1, 0.5, 1, "あ。"},
		{"emphasis", "7d36684181d0cdcb9d0ad8e421b5d69cea6dbbe25fa9c710a0ea48e87cd4a55b", 49200, 1, 1, 1, 1.5, "あ。"},
		{"mixed", "15ede70d53c1e34d1e1e2e0bd63d2c081be1432d50100e423486db2277e8a587", 117113, 0.5, 1.2, 0.5, 1.5, "こんにちは。"},
		{"high", "769a793e9b269afd718380f641c01d69ad07c98f64c1f4877dfefb0e6b5a6145", 45120, 2, 2, 2, 0.01, "あ。"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want, err := os.ReadFile(filepath.Join("..", "..", "private", "control-"+tc.name+"-original.wav"))
			if os.IsNotExist(err) {
				t.Skip("private original controlled WAV unavailable")
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(want) != 44+tc.samples*2 || fmt.Sprintf("%x", sha256.Sum256(want)) != tc.hash {
				t.Fatal("original controlled WAV reference differs")
			}
			if e == nil {
				e, err = Open(Options{AssetsDir: "../../assets", Runtime: filepath.Join(t.TempDir(), "missing-runtime")})
				if err != nil {
					t.Fatal(err)
				}
			}
			got, err := e.Synthesize(context.Background(), engine.Request{Text: tc.text, Voice: VoiceAoi, Rate: tc.rate, Pitch: tc.pitch, Volume: tc.volume, Emphasis: tc.emphasis})
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("complete controlled WAV differs: got SHA256 %x, want %s", sha256.Sum256(got), tc.hash)
			}
			t.Logf("Original controlled text-to-WAV: %d PCM samples, complete file bit-identical", tc.samples)
		})
	}
}
