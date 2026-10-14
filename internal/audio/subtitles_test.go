package audio

import (
	"strings"
	"testing"
	"time"

	"github.com/a-kaibu/aoikane/internal/engine"
)

func TestSubtitles(t *testing.T) {
	boundary := 1234567 * time.Microsecond
	cues := []engine.Cue{{Text: "茜 & <葵>\nです。", End: boundary}, {Text: "続き。", Start: boundary, End: time.Hour + time.Millisecond}}
	for _, format := range []string{"srt", "vtt"} {
		b, err := Subtitles(cues, format)
		if err != nil {
			t.Fatal(err)
		}
		separator := ","
		if format == "vtt" {
			separator = "."
			if !strings.HasPrefix(string(b), "WEBVTT\n\n") {
				t.Fatal(string(b))
			}
		}
		for _, want := range []string{"茜 &amp; &lt;葵&gt; です。", "00:00:01" + separator + "235", "01:00:00" + separator + "001"} {
			if !strings.Contains(string(b), want) {
				t.Fatalf("missing %q in %s", want, b)
			}
		}
		if strings.Count(string(b), "00:00:01"+separator+"235") != 2 {
			t.Fatal("contiguous cues must share the same rounded boundary")
		}
	}
}

func TestInvalidSubtitles(t *testing.T) {
	for _, cues := range [][]engine.Cue{
		{{Text: "a", Start: -time.Second, End: time.Second}},
		{{Text: "a", End: time.Second}, {Text: "b", Start: time.Millisecond, End: 2 * time.Second}},
		{{Text: " ", End: time.Second}},
		{{Text: "a", End: time.Microsecond}},
	} {
		if _, err := Subtitles(cues, "srt"); err == nil {
			t.Fatal("accepted invalid cues")
		}
	}
	if _, err := Subtitles(nil, "other"); err == nil {
		t.Fatal("accepted invalid format")
	}
}

func TestWholeAudioCue(t *testing.T) {
	w, err := engine.WAV(make([]byte, 96000), 48000)
	if err != nil {
		t.Fatal(err)
	}
	cues, err := WholeAudioCue(w, "文章。")
	if err != nil {
		t.Fatal(err)
	}
	if len(cues) != 1 || cues[0].End != time.Second || cues[0].Text != "文章。" {
		t.Fatal(cues)
	}
	if _, err := WholeAudioCue([]byte("invalid"), "文章。"); err == nil {
		t.Fatal("accepted invalid WAV")
	}
}
