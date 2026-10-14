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

func TestRemoteSubtitles(t *testing.T) {
	wav, err := engine.WAV(make([]byte, 96000), 48000)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request engine.Request
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		if request.Format != "wav" {
			t.Errorf("timing requires WAV, got %q", request.Format)
		}
		w.Write(wav)
	}))
	defer server.Close()
	for _, format := range []string{"wav", "mp3"} {
		t.Run(format, func(t *testing.T) {
			dir := t.TempDir()
			out, srt, vtt := filepath.Join(dir, "speech."+format), filepath.Join(dir, "speech.srt"), filepath.Join(dir, "speech.vtt")
			if err := synth([]string{"--url", server.URL, "--text", "茜です。", "-o", out, "--srt", srt, "--vtt", vtt}); err != nil {
				t.Fatal(err)
			}
			b, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}
			if format == "wav" && !bytes.Equal(b, wav) {
				t.Fatal("subtitles changed audio")
			}
			if format == "mp3" && (len(b) < 4 || b[0] != 0xff || b[1]&0xe0 != 0xe0) {
				t.Fatal("invalid MP3")
			}
			for path, want := range map[string]string{srt: "00:00:00,000 --> 00:00:01,000", vtt: "00:00:00.000 --> 00:00:01.000"} {
				b, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(b), want) || !strings.Contains(string(b), "茜です。") {
					t.Fatal(string(b))
				}
			}
		})
	}
}

func TestSubtitleOutputPaths(t *testing.T) {
	for _, paths := range [][]string{{"speech.wav", "./speech.wav", ""}, {"speech.wav", "same", "same"}, {"speech.wav", "-", ""}} {
		if err := validateOutputPaths(paths...); err == nil {
			t.Fatal(paths)
		}
	}
	if err := validateOutputPaths("-", "speech.srt", "speech.vtt"); err != nil {
		t.Fatal(err)
	}
}
