package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/a-kaibu/aoikane/internal/engine"
)

func TestDefaultSynthesisNames(t *testing.T) {
	wav, err := engine.WAV(make([]byte, 96000), 48000)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request engine.Request
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		if request.Text != "茜です。" || request.Format != "wav" {
			t.Errorf("unexpected request: %+v", request)
		}
		w.Write(wav)
	}))
	defer server.Close()
	for _, kind := range []string{"positional", "file", "input", "output", "o", "mp3", "legacy", "stdin"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "台本.txt")
			if err := os.WriteFile(input, []byte("茜です。"), 0600); err != nil {
				t.Fatal(err)
			}
			base := filepath.Join(dir, "台本")
			args := []string{input, "--url", server.URL, "--srt", "--vtt"}
			switch kind {
			case "file", "input":
				args = append([]string{"--" + kind, input}, args[1:]...)
			case "output", "o":
				base = filepath.Join(dir, "custom")
				args = append(args, "--"+kind, base+".wav")
			case "mp3":
				args = append(args, "--format", "mp3")
			case "legacy":
				args = append([]string{"synth"}, args...)
			case "stdin":
				t.Chdir(dir)
				t.Setenv("AOIKANE_URL", server.URL)
				stdin, err := os.Open(input)
				if err != nil {
					t.Fatal(err)
				}
				defer stdin.Close()
				previous := os.Stdin
				os.Stdin = stdin
				defer func() { os.Stdin = previous }()
				args = nil
				base = filepath.Join(dir, "speech")
			}
			if err := run(args); err != nil {
				t.Fatal(err)
			}
			ext := ".wav"
			if kind == "mp3" {
				ext = ".mp3"
			}
			b, err := os.ReadFile(base + ext)
			if err != nil {
				t.Fatal(err)
			}
			if ext == ".wav" && !bytes.Equal(b, wav) {
				t.Fatal("audio changed")
			}
			for _, ext := range []string{".srt", ".vtt"} {
				if kind == "stdin" {
					continue
				}
				b, err := os.ReadFile(base + ext)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(b), "茜です。") {
					t.Fatal(string(b))
				}
			}
		})
	}
}

func TestHelpAndVersion(t *testing.T) {
	for _, args := range [][]string{{"help"}, {"--help"}, {"-h"}, {"version"}, {"--version"}, {"-version"}, {"-v"}} {
		if err := run(args); err != nil {
			t.Fatal(err)
		}
	}
}

func TestBareInvocationReadsStdin(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "stdin.txt")
	if err := os.WriteFile(input, nil, 0600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(input)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	old := os.Stdin
	os.Stdin = f
	defer func() { os.Stdin = old }()
	if err := run(nil); err == nil || !strings.Contains(err.Error(), "text") {
		t.Fatalf("empty stdin must be validated as synthesis input: %v", err)
	}
}

func TestVoiceAliasesReachSynthesis(t *testing.T) {
	wav, err := engine.WAV(make([]byte, 960), 48000)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, id string }{
		{"aoi", "KotonohaAoi_ns_48"},
		{"akane", "KotonohaAkane_ns_48"},
		{"葵", "KotonohaAoi_ns_48"},
		{"茜", "KotonohaAkane_ns_48"},
		{"KotonohaAkane_ns_48", "KotonohaAkane_ns_48"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request engine.Request
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
				}
				if request.Voice != tc.id {
					t.Errorf("voice = %q; want %q", request.Voice, tc.id)
				}
				w.Write(wav)
			}))
			defer server.Close()
			if err := run([]string{"--text", "こんにちは。", "--voice", tc.name, "--url", server.URL, "-o", filepath.Join(t.TempDir(), "speech.wav")}); err != nil {
				t.Fatal(err)
			}
		})
	}
}
