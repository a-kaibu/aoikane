//go:build (windows || darwin || linux) && (amd64 || arm64)

package portable

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"testing"

	"github.com/a-kaibu/aoikane/internal/engine"
)

type domainChunkContextReference struct {
	Text  string `json:"text"`
	Cases []struct {
		Voice    string   `json:"voice"`
		Readings []string `json:"readings"`
		WAVBytes int      `json:"wav_bytes"`
		WAVSHA   string   `json:"wav_sha256"`
	} `json:"cases"`
}

func loadDomainChunkContextReference(t *testing.T) domainChunkContextReference {
	t.Helper()
	raw, err := os.ReadFile("testdata/domain_chunk_context_original.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref domainChunkContextReference
	if err := json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.Text == "" || len(ref.Cases) != 2 {
		t.Fatal("original text reference is incomplete")
	}
	return ref
}

func TestDomainChunkContextReadingOriginal(t *testing.T) {
	ref := loadDomainChunkContextReference(t)
	e, err := Open(Options{AssetsDir: "../../assets"})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	for _, tc := range ref.Cases {
		voice := VoiceAoi
		if tc.Voice == "akane" {
			voice = VoiceAkane
		} else if tc.Voice != "aoi" {
			t.Fatalf("unknown reference voice %q", tc.Voice)
		}
		t.Run(tc.Voice, func(t *testing.T) {
			got, err := e.ReadingVoice(ref.Text, voice)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, tc.Readings) {
				t.Fatalf("reading differs: got %q; want %q", got, tc.Readings)
			}
		})
	}
}

func TestDomainChunkContextWAVOriginal(t *testing.T) {
	if os.Getenv("AOIKANE_DOMAIN_CHUNK_WAV_AUDIT") != "1" {
		t.Skip("set AOIKANE_DOMAIN_CHUNK_WAV_AUDIT=1 to compare original WAVs")
	}
	ref := loadDomainChunkContextReference(t)
	e, err := Open(Options{AssetsDir: "../../assets", OriginalThreads: 6})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	for _, tc := range ref.Cases {
		voice := VoiceAoi
		if tc.Voice == "akane" {
			voice = VoiceAkane
		}
		t.Run(tc.Voice, func(t *testing.T) {
			got, err := e.Synthesize(context.Background(), engine.Request{Text: ref.Text, Voice: voice})
			if err != nil {
				t.Fatal(err)
			}
			digest := fmt.Sprintf("%x", sha256.Sum256(got))
			if len(got) != tc.WAVBytes || digest != tc.WAVSHA {
				t.Fatalf("whole WAV differs: got %d bytes %s; original %d bytes %s", len(got), digest, tc.WAVBytes, tc.WAVSHA)
			}
			t.Logf("whole original WAV identical: %q, %d bytes", ref.Text, len(got))
		})
	}
}
