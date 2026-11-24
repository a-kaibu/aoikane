// Package english adapts Pocket TTS to the Aoikane synthesis API.
package english

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/a-kaibu/aoikane/internal/assetfs"
	"github.com/a-kaibu/aoikane/internal/engine"
	"github.com/a-kaibu/aoikane/internal/english/pockettts"
)

const (
	DefaultDir   = "assets/english"
	DefaultVoice = "en-alba"
)

var voices = [...]engine.Voice{
	{ID: DefaultVoice, Name: "Alba (English)", SampleRate: pockettts.SampleRate},
	{ID: "en-aoi", Name: "Aoi (English)", SampleRate: pockettts.SampleRate},
	{ID: "en-akane", Name: "Akane (English)", SampleRate: pockettts.SampleRate},
}

func VoicePath(voice string) string {
	for _, spec := range voices {
		if voice == spec.ID {
			return filepath.Join("voices", voice[len("en-"):]+".safetensors")
		}
	}
	return ""
}

type Engine struct {
	mu     sync.Mutex
	model  *pockettts.Engine
	voices map[string]*pockettts.Voice
}

func OpenModel(dir string) (*Engine, error) {
	weights, err := assetfs.ReadFile(filepath.Join(dir, "model.safetensors"))
	if err != nil {
		return nil, fmt.Errorf("English model (run scripts/prepare-english.py): %w", err)
	}
	tokenizer, err := assetfs.ReadFile(filepath.Join(dir, "tokenizer.model"))
	if err != nil {
		return nil, err
	}
	model, err := pockettts.Load(weights, tokenizer)
	if err != nil {
		return nil, err
	}
	return &Engine{model: model, voices: make(map[string]*pockettts.Voice)}, nil
}

func Open(dir string) (*Engine, error) {
	e, err := OpenModel(dir)
	if err != nil {
		return nil, err
	}
	for _, voice := range voices {
		if err := e.LoadVoice(dir, voice.ID); err != nil && !assetfs.NotExist(err) {
			e.Close()
			return nil, err
		}
	}
	if len(e.voices) == 0 {
		e.Close()
		return nil, fmt.Errorf("no English voices in %s", filepath.Join(dir, "voices"))
	}
	return e, nil
}

// LoadVoice adds one precomputed voice. The browser calls this after fetching it.
func (e *Engine) LoadVoice(dir, id string) error {
	path := VoicePath(id)
	if path == "" {
		return fmt.Errorf("unknown English voice %q", id)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.model == nil {
		return fmt.Errorf("English model is closed")
	}
	if e.voices[id] != nil {
		return nil
	}
	data, err := assetfs.ReadFile(filepath.Join(dir, path))
	if err != nil {
		return err
	}
	voice, err := e.model.ParseVoice(data)
	if err != nil {
		return fmt.Errorf("English voice %s: %w", id, err)
	}
	e.voices[id] = voice
	return nil
}

func (e *Engine) Voices() []engine.Voice {
	e.mu.Lock()
	defer e.mu.Unlock()
	list := make([]engine.Voice, 0, len(e.voices))
	for _, voice := range voices {
		if e.voices[voice.ID] != nil {
			list = append(list, voice)
		}
	}
	return list
}

func (e *Engine) Synthesize(ctx context.Context, request engine.Request) ([]byte, error) {
	wav, _, err := e.SynthesizeTimed(ctx, request)
	return wav, err
}

func (e *Engine) SynthesizeTimed(ctx context.Context, r engine.Request) ([]byte, []engine.Cue, error) {
	if err := validateRequest(&r); err != nil {
		return nil, nil, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	voice := e.voices[r.Voice]
	if voice == nil {
		return nil, nil, fmt.Errorf("English voice %q is not installed", r.Voice)
	}

	var pcm []byte
	var cues []engine.Cue
	const pcmLimit = 256 << 20
	duration := func() time.Duration { return time.Duration(len(pcm)/2) * time.Second / pockettts.SampleRate }
	err := e.model.Synthesize(ctx, r.Text, voice, e.model.DefaultOptions(), func(text string, samples []float32) error {
		if len(samples) > (pcmLimit-len(pcm))/2 {
			return fmt.Errorf("synthesized audio exceeds 256 MiB")
		}
		start := duration()
		var err error
		pcm, err = appendPCM(pcm, samples, r.Volume)
		if err != nil {
			return err
		}

		cues = append(cues, engine.Cue{Text: text, Start: start, End: duration()})
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	wav, err := engine.WAV(pcm, pockettts.SampleRate)
	return wav, cues, err
}

func validateRequest(r *engine.Request) error {
	r.Defaults()
	if err := r.Validate(); err != nil {
		return err
	}
	if r.Rate != 1 || r.Pitch != 1 || r.Emphasis != 1 {
		return fmt.Errorf("English voices support volume and require default rate, pitch and emphasis")
	}
	return nil
}

func appendPCM(pcm []byte, samples []float32, volume float64) ([]byte, error) {
	for _, sample := range samples {
		v := float64(sample) * volume
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, fmt.Errorf("English synthesis produced a non-finite sample")
		}
		pcm = binary.LittleEndian.AppendUint16(pcm, uint16(int16(math.Round(min(1, max(-1, v))*32767))))
	}
	return pcm, nil
}

func (e *Engine) Prepare(r engine.Request) ([]pockettts.Chunk, error) {
	if err := validateRequest(&r); err != nil {
		return nil, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.voices[r.Voice] == nil {
		return nil, fmt.Errorf("English voice %q is not installed", r.Voice)
	}
	return e.model.Prepare(r.Text)
}

func (e *Engine) SynthesizeChunk(ctx context.Context, r engine.Request, chunk pockettts.Chunk, index int) ([]byte, []engine.Cue, error) {
	if err := validateRequest(&r); err != nil {
		return nil, nil, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	voice := e.voices[r.Voice]
	if voice == nil {
		return nil, nil, fmt.Errorf("English voice %q is not installed", r.Voice)
	}
	samples, err := e.model.SynthesizeChunk(ctx, chunk, voice, e.model.DefaultOptions(), index)
	if err != nil {
		return nil, nil, err
	}
	pcm, err := appendPCM(make([]byte, 0, len(samples)*2), samples, r.Volume)
	if err != nil {
		return nil, nil, err
	}
	wav, err := engine.WAV(pcm, pockettts.SampleRate)
	return wav, []engine.Cue{{Text: strings.TrimSpace(chunk.Text), End: time.Duration(len(samples)) * time.Second / pockettts.SampleRate}}, err
}

func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.model != nil {
		e.model.Close()
		e.model = nil
		clear(e.voices)
	}
	return nil
}
