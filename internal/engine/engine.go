package engine

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"
)

type Voice struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	SampleRate int    `json:"sample_rate"`
}

type Request struct {
	Format   string  `json:"format,omitempty"`
	Text     string  `json:"text"`
	Voice    string  `json:"voice,omitempty"`
	Rate     float64 `json:"rate,omitempty"`
	Pitch    float64 `json:"pitch,omitempty"`
	Volume   float64 `json:"volume,omitempty"`
	Emphasis float64 `json:"emphasis,omitempty"`
}

func (r *Request) Defaults() {
	if r.Format == "" {
		r.Format = "wav"
	}
	if r.Rate == 0 {
		r.Rate = 1
	}
	if r.Pitch == 0 {
		r.Pitch = 1
	}
	if r.Volume == 0 {
		r.Volume = 1
	}
	if r.Emphasis == 0 {
		r.Emphasis = 1
	}
}

func (r Request) Validate() error {
	if r.Format != "" && r.Format != "wav" && r.Format != "mp3" {
		return errors.New("format must be wav or mp3")
	}
	if !utf8.ValidString(r.Text) || strings.TrimSpace(r.Text) == "" || strings.ContainsRune(r.Text, 0) {
		return errors.New("text must be nonempty UTF-8 without NUL characters")
	}
	if len(r.Text) > 65536 {
		return errors.New("text exceeds 65536 UTF-8 bytes")
	}
	for _, control := range []struct {
		name            string
		value, min, max float64
	}{
		{"rate", r.Rate, 0.5, 4},
		{"pitch", r.Pitch, 0.5, 2},
		{"volume", r.Volume, 0, 2},
		{"emphasis", r.Emphasis, 0, 2},
	} {
		if math.IsNaN(control.value) || math.IsInf(control.value, 0) || control.value < control.min || control.value > control.max {
			return fmt.Errorf("%s must be between %g and %g", control.name, control.min, control.max)
		}
	}
	return nil
}

type Synthesizer interface {
	Voices() []Voice
	Synthesize(context.Context, Request) ([]byte, error)
	Close() error
}

// Cue covers a sentence in the synthesized audio, including its pauses.
type Cue struct {
	Text       string
	Start, End time.Duration
}

// TimedSynthesizer returns timings from the same synthesis as the WAV.
type TimedSynthesizer interface {
	SynthesizeTimed(context.Context, Request) ([]byte, []Cue, error)
}
