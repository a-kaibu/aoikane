package audio

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/a-kaibu/aoikane/internal/engine"
	"github.com/go-audio/wav"
)

// WholeAudioCue is used when a backend provides audio without alignments.
func WholeAudioCue(data []byte, text string) ([]engine.Cue, error) {
	d := wav.NewDecoder(bytes.NewReader(data))
	b, err := d.FullPCMBuffer()
	if err != nil {
		return nil, fmt.Errorf("subtitle WAV: %w", err)
	}
	if b == nil || b.Format == nil || b.Format.SampleRate <= 0 || b.Format.NumChannels <= 0 || len(b.Data) == 0 {
		return nil, fmt.Errorf("subtitle WAV has no audio")
	}
	end := time.Duration(len(b.Data)/b.Format.NumChannels) * time.Second / time.Duration(b.Format.SampleRate)
	return []engine.Cue{{Text: text, End: end}}, nil
}

// Subtitles serializes actual audio cue boundaries to UTF-8 SRT or WebVTT.
func Subtitles(cues []engine.Cue, format string) ([]byte, error) {
	if format != "srt" && format != "vtt" {
		return nil, fmt.Errorf("subtitle format must be srt or vtt")
	}
	var out strings.Builder
	if format == "vtt" {
		out.WriteString("WEBVTT\n\n")
	}
	previous := time.Duration(0)
	for i, c := range cues {
		if c.Start < previous || c.End <= c.Start || strings.TrimSpace(c.Text) == "" {
			return nil, fmt.Errorf("invalid subtitle cue %d", i+1)
		}
		start, end := c.Start.Round(time.Millisecond).Milliseconds(), c.End.Round(time.Millisecond).Milliseconds()
		if end <= start {
			return nil, fmt.Errorf("subtitle cue %d is shorter than timestamp precision", i+1)
		}
		text := strings.Join(strings.Fields(c.Text), " ")
		text = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(text)
		fmt.Fprintf(&out, "%d\n%s --> %s\n%s\n\n", i+1, timestamp(start, format), timestamp(end, format), text)
		previous = c.End
	}
	return []byte(out.String()), nil
}

func timestamp(ms int64, format string) string {
	separator := ","
	if format == "vtt" {
		separator = "."
	}
	return fmt.Sprintf("%02d:%02d:%02d%s%03d", ms/3600000, ms/60000%60, ms/1000%60, separator, ms%1000)
}
