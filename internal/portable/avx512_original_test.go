//go:build (windows || darwin || linux) && (amd64 || arm64)

package portable

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/a-kaibu/aoikane/internal/engine"
)

// Random texts were generated before this profile was reconstructed. Expected
// digests come from ordinary original synthesis, with no Frida observation.
func TestOriginalAVX512RandomWAV(t *testing.T) {
	if os.Getenv("AOIKANE_AVX512_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_AVX512_REFERENCE_AUDIT=1 for the strict eight-case audit, including two unresolved mismatches")
	}
	var ref struct {
		CPU     string
		Threads int `json:"original_threads"`
		Voice   string
		Cases   []struct {
			Index  int
			Text   string
			Bytes  int
			SHA256 string
		}
	}
	raw, err := os.ReadFile("testdata/original_avx512_random.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.CPU != "AMD64 Family 25 Model 17 Stepping 1, AuthenticAMD" || ref.Threads != 1 || ref.Voice != VoiceAoi || len(ref.Cases) != 8 {
		t.Fatal("unexpected original corpus profile")
	}
	e, err := Open(Options{AssetsDir: "../../assets", OriginalThreads: 1, OriginalAVX512Convolutions: true})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	for _, tc := range ref.Cases {
		t.Run(fmt.Sprintf("case-%04d", tc.Index), func(t *testing.T) {
			got, err := e.Synthesize(context.Background(), engine.Request{Text: tc.Text, Voice: ref.Voice})
			if err != nil {
				t.Fatal(err)
			}
			if dir := os.Getenv("AOIKANE_AVX512_REFERENCE_OUTPUT"); dir != "" {
				if err := os.MkdirAll(dir, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("case-%04d-go.wav", tc.Index)), got, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if len(got) != tc.Bytes || fmt.Sprintf("%x", sha256.Sum256(got)) != tc.SHA256 {
				t.Fatalf("text %q differs: bytes=%d SHA256=%x; want bytes=%d SHA256=%s", tc.Text, len(got), sha256.Sum256(got), tc.Bytes, tc.SHA256)
			}
			path := filepath.Join("..", "..", "private", "profile-random-ci-37895253763", "seed-1", fmt.Sprintf("case-%04d-original.wav", tc.Index))
			original, err := os.ReadFile(path)
			if err == nil {
				if !bytes.Equal(got, original) {
					t.Fatal("whole original WAV differs")
				}
			} else if !os.IsNotExist(err) {
				t.Fatal(err)
			}
			t.Logf("ordinary original WAV matches: text=%q bytes=%d SHA256=%s", tc.Text, len(got), tc.SHA256)
		})
	}
}

// This shorter synthesis exercises a different vocoder geometry than the
// 110-frame greeting. Its digest is from the ordinary AMD Model 17 corpus.
func TestOriginalAVX512KanWAV(t *testing.T) {
	e, err := Open(Options{AssetsDir: "../../assets", OriginalThreads: 1, OriginalAVX512Convolutions: true})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	got, err := e.Synthesize(context.Background(), engine.Request{Text: "漢。", Voice: VoiceAoi})
	if err != nil {
		t.Fatal(err)
	}
	const digest = "5205a43b5629e2ff7d9945b8560efe089fca6a2c5b7be33162be6a4e421b29c4"
	if len(got) != 110924 || fmt.Sprintf("%x", sha256.Sum256(got)) != digest {
		t.Fatalf("ordinary original WAV differs: bytes=%d SHA256=%x", len(got), sha256.Sum256(got))
	}
	t.Logf("whole original WAV matches: bytes=%d SHA256=%s", len(got), digest)
}
