//go:build (windows || darwin || linux) && (amd64 || arm64)

package portable

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/a-kaibu/aoikane/internal/engine"
)

// Compare the complete production text-to-WAV path against independently
// generated original files, without feeding native model inputs to Go.
func TestOriginalWAV(t *testing.T) {
	var e *Engine
	t.Cleanup(func() {
		if e != nil {
			e.Close()
		}
	})
	for _, tc := range []struct {
		name, text, file, hash string
		samples                int
	}{
		{"dependency-reading", "窓を開ける。", "reading-window-original.wav", "a78010ee253221f1c5ef48f4750652ab72ae06cf217e84f4f958bdbe4a2f27da", 86556},
		{"short", "あ。", "short-original-all-go-check.wav", "2bd9a6674b3ac583fdeb88aca32814826b24d69882bfe6821d8fd87d7077738d", 49200},
		{"four-phones", "あい。", "four-phones-original.wav", "bbb68bc21f21ef340ed1c689f386d82da8464fcf7d653b3b28e57e9b23dfd0ed", 55680},
		{"seven-phones", "あいうえお。", "seven-phones-original.wav", "3b7be89fae9ca67a9465c4ded780eec34f23cb92cf8025e8748b877e382cbab7", 83244},
		{"eight-phones", "あいうえおあ。", "eight-phones-original.wav", "ee4972d1204f27bc3ce958fcb6d7678874907bd8d939d2e73e69555b07fbc4a1", 89301},
		{"common-sentences", "こんにちは。今日はいい天気ですね。これは音声合成のテストです。", "common-sentences-original.wav", "25da3fff4f4d917a0a64662ec80e21dddf458f35fdf8f145a6ee5a43030e7990", 365221},
		{"long-mora", "ーカナです。ンーカナです。ッーカナです。アーです。アーーです。", "long-mora-original-all-go-check.wav", "9e14c6655f3efe4af71985a5ebb4c13e1f3177090755cfd04958ddf60e28ecd5", 371946},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want, err := os.ReadFile(filepath.Join("..", "..", "private", tc.file))
			if os.IsNotExist(err) {
				t.Skip("private original WAV unavailable")
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(want) != 44+tc.samples*2 || fmt.Sprintf("%x", sha256.Sum256(want)) != tc.hash {
				t.Fatal("original WAV reference differs")
			}
			if e == nil {
				e, err = Open(Options{AssetsDir: "../../assets", Runtime: filepath.Join(t.TempDir(), "missing-runtime")})
				if err != nil {
					t.Fatal(err)
				}
			}
			got, err := e.Synthesize(context.Background(), engine.Request{Text: tc.text, Voice: VoiceAoi})
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("complete WAV differs: got SHA256 %x, want %s", sha256.Sum256(got), tc.hash)
			}
			t.Logf("Original text-to-WAV: %d PCM samples, complete file bit-identical", tc.samples)
		})
	}
}

// Entire WAV digests from ordinary original synthesis: CI run 37890647782
// uses one thread, the local original uses six. Only text enters Go synthesis.
func TestOriginalThreadProfilesWAV(t *testing.T) {
	for _, tc := range []struct {
		threads int
		text    string
		bytes   int
		hash    string
	}{
		{1, "こんにちは。", 157964, "516c51009cdd5de09d5eb4e70d9cd1235445b729528d755f7c73ac3eaea699d2"},
		{6, "こんにちは。", 157964, "d22ec328e5937a17f1f8159db0d6381d4e81826ffe57e712ff977c6f6539ff93"},
		{1, "漢。", 110924, "5d7b00b7b10edb89ca667431b17cf9506376810a1ccc256ba2fdf7c8b0681383"},
	} {
		t.Run(fmt.Sprintf("threads-%d/%s", tc.threads, tc.text), func(t *testing.T) {
			e, err := Open(Options{AssetsDir: "../../assets", OriginalThreads: tc.threads})
			if err != nil {
				t.Fatal(err)
			}
			defer e.Close()
			got, err := e.Synthesize(context.Background(), engine.Request{Text: tc.text, Voice: VoiceAoi})
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != tc.bytes || fmt.Sprintf("%x", sha256.Sum256(got)) != tc.hash {
				t.Fatalf("original whole WAV differs: bytes=%d SHA256=%x", len(got), sha256.Sum256(got))
			}
			t.Logf("complete original WAV matches: %d bytes, SHA256=%s", len(got), tc.hash)
		})
	}
}

// Reference: ordinary, uninstrumented synthesis on AMD Family 26 Model 2,
// CI 37927408109 replica 1. Only Japanese text enters the Go engine.
func TestOriginalAVX512ProfileWAV(t *testing.T) {
	e, err := Open(Options{AssetsDir: "../../assets", OriginalThreads: 1, OriginalAVX512Convolutions: true})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	got, err := e.Synthesize(context.Background(), engine.Request{Text: "こんにちは。", Voice: VoiceAoi})
	if err != nil {
		t.Fatal(err)
	}
	const digest = "59d1eec127cc579fd751a025b9d1d85122996436e12233332a5f8bf8bb1af193"
	if len(got) != 157964 || fmt.Sprintf("%x", sha256.Sum256(got)) != digest {
		t.Fatalf("original AVX512 whole WAV differs: bytes=%d SHA256=%x", len(got), sha256.Sum256(got))
	}
	path := filepath.Join("..", "..", "private", "cpu-numeric-ci-37927408109", "cpu-original-replica-1", "sentence", "case-0000-original.wav")
	want, err := os.ReadFile(path)
	if err == nil {
		if !bytes.Equal(got, want) {
			t.Fatal("ordinary original WAV differs")
		}
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
	t.Logf("complete original WAV matches: %d bytes, SHA256=%s", len(got), digest)
}
func TestRejectUnsupportedAVX512Profile(t *testing.T) {
	if _, err := Open(Options{AssetsDir: t.TempDir(), OriginalThreads: 6, OriginalAVX512Convolutions: true}); err == nil || !strings.Contains(err.Error(), "AVX512 convolution profile requires one original thread") {
		t.Fatalf("unsupported profile was not rejected before asset loading: %v", err)
	}
}
