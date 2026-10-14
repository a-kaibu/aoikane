package portable

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/a-kaibu/aoikane/internal/audio"
	"github.com/a-kaibu/aoikane/internal/engine"
)

// Replay the whole original WAV, including pauses, with timing enabled.
func TestTimedSynthesisOriginalWAV(t *testing.T) {
	if os.Getenv("AOIKANE_SUBTITLE_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_SUBTITLE_REFERENCE_AUDIT=1 for original WAV replay")
	}
	e, err := Open(Options{AssetsDir: "../../assets"})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	for _, voice := range []string{VoiceAkane, VoiceAoi} {
		t.Run(voice, func(t *testing.T) {
			wav, cues, err := e.SynthesizeTimed(context.Background(), engine.Request{Text: "0京秒1。", Voice: voice})
			if err != nil {
				t.Fatal(err)
			}
			if len(wav) != 175184 || fmt.Sprintf("%x", sha256.Sum256(wav)) != "720f38c865d54aef04297d01ebf330c77901e0566d034b7a479f3fa66f2516cf" {
				t.Fatal("whole WAV differs from original")
			}
			end := time.Duration((len(wav)-44)/2) * time.Second / 48000
			if len(cues) != 1 || cues[0].Text != "0京秒1。" || cues[0].Start != 0 || cues[0].End != end {
				t.Fatal(cues)
			}
			for _, format := range []string{"srt", "vtt"} {
				if _, err := audio.Subtitles(cues, format); err != nil {
					t.Fatal(err)
				}
			}
			t.Logf("whole original WAV identical, %d bytes; cue ends at %v", len(wav), end)
		})
	}
}

func TestTimedSentencesOriginalWAV(t *testing.T) {
	if os.Getenv("AOIKANE_SUBTITLE_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_SUBTITLE_REFERENCE_AUDIT=1 for original WAV replay")
	}
	e, err := Open(Options{AssetsDir: "../../assets"})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	for _, ref := range []struct {
		voice, hash string
		size        int
	}{
		{VoiceAkane, "46dc082bb156ea3c63731907126273d5fdd91f007c0516ad82990452c2632d4f", 274604},
		{VoiceAoi, "b57d0fe7f0f81312421bce04b360bf7698f4b0c7b7b0bab84966c328e5cbee19", 273644},
	} {
		t.Run(ref.voice, func(t *testing.T) {
			w, cues, err := e.SynthesizeTimed(context.Background(), engine.Request{Text: "あ。 。い。", Voice: ref.voice})
			if err != nil {
				t.Fatal(err)
			}
			if len(w) != ref.size || fmt.Sprintf("%x", sha256.Sum256(w)) != ref.hash {
				t.Fatal("whole multi-sentence WAV differs from original")
			}
			if len(cues) != 3 || cues[0].Text != "あ。" || cues[1].Text != "。" || cues[2].Text != "い。" {
				t.Fatal(cues)
			}
			if cues[0].Start != 0 || cues[0].End != cues[1].Start || cues[1].End != cues[2].Start || cues[2].End != time.Duration((len(w)-44)/2)*time.Second/48000 {
				t.Fatal(cues)
			}
			for _, format := range []string{"srt", "vtt"} {
				if _, err := audio.Subtitles(cues, format); err != nil {
					t.Fatal(err)
				}
			}
			t.Logf("whole original WAV identical, %d bytes; cues: %+v", len(w), cues)
		})
	}
}
