//go:build (windows || darwin || linux) && (amd64 || arm64)

// Package portable assembles the reconstructed text frontend, the neural
// acoustic model and the vocoder into an engine.Synthesizer that never calls
// the original engine. Standard Japanese (葵) and Kansai Japanese (茜) use separate frontends.
package portable

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/a-kaibu/aoikane/internal/cabocha"
	"github.com/a-kaibu/aoikane/internal/engine"
	"github.com/a-kaibu/aoikane/internal/frontend"
	"github.com/a-kaibu/aoikane/internal/numeral"
	"github.com/a-kaibu/aoikane/internal/onnx"
	"github.com/a-kaibu/aoikane/internal/phrase"
	"github.com/a-kaibu/aoikane/internal/synthesis"
)

// VoiceAoi is the voice ID of the supported standard voice.
const VoiceAoi = "KotonohaAoi_ns_48"

// VoiceAkane is the Kansai voice ID.
const VoiceAkane = "KotonohaAkane_ns_48"

// Options locate the assets (the layout of the repository's assets directory).
type Options struct {
	AssetsDir string
	// Runtime is retained for source compatibility and is no longer used.
	Runtime string
	// OriginalThreads selects numerical partitioning (1 or 6). Zero retains
	// the existing six-thread profile; it does not configure Go workers.
	OriginalThreads int
	// OriginalAVX512Convolutions selects the audited one-thread convolution order.
	// It runs in pure Go and does not require AVX512 hardware.
	OriginalAVX512Convolutions bool
}

func (o *Options) defaults() {
	if o.AssetsDir == "" {
		o.AssetsDir = "assets"
	}
	if o.OriginalThreads == 0 {
		o.OriginalThreads = 6
	}
}

// Engine is the portable synthesizer.
type Engine struct {
	frontend      *phrase.Frontend
	akane         *phrase.Frontend
	akaneControls synthesis.Controls
	predict       *phrase.BoundModel
	encoder       *frontend.MoraEncoder
	synth         *synthesis.Engine
	controls      synthesis.Controls
}

// Open loads every asset; it fails if any is missing.
func Open(o Options) (*Engine, error) {
	o.defaults()
	if o.OriginalThreads != 1 && o.OriginalThreads != 6 {
		return nil, fmt.Errorf("original thread profile %d is not reconstructed", o.OriginalThreads)
	}
	if o.OriginalAVX512Convolutions && o.OriginalThreads != 1 {
		return nil, fmt.Errorf("AVX512 convolution profile requires one original thread")
	}
	a := func(parts ...string) string { return filepath.Join(append([]string{o.AssetsDir}, parts...)...) }
	analyzer, err := frontend.OpenAnalyzer(a("dict"))
	if err != nil {
		return nil, fmt.Errorf("dictionary: %w", err)
	}
	morphology, err := frontend.OpenMorphology(a("data", "morphology.json"))
	if err != nil {
		return nil, err
	}
	codec, err := frontend.OpenCP932Codec(a("data", "cp932.json"))
	if err != nil {
		return nil, err
	}
	reading, err := openRules(a("rules", "reading-rules-standard.txt"), frontend.ParseReadingRules)
	if err != nil {
		return nil, err
	}
	bound, err := openRules(a("rules", "bound-rules-standard.txt"), phrase.ParseBoundRules)
	if err != nil {
		return nil, err
	}
	numerals, err := openRules(a("rules", "bancharcx-standard.txt"), numeral.ParseRules)
	if err != nil {
		return nil, err
	}
	rendakuAssets, err := openRules(a("rules", "rendaku.json"), phrase.ParseRendakuAssets)
	if err != nil {
		return nil, err
	}
	rendaku, err := rendakuAssets.Rules("standard")
	if err != nil {
		return nil, err
	}
	cut, err := openRules(a("rules", "bancut-standard.json"), phrase.ParseCutRules)
	if err != nil {
		return nil, err
	}
	unknownKana, err := frontend.OpenUnknownKanaRules(a("rules", "unknown-kana.json"))
	if err != nil {
		return nil, err
	}
	unknownAccent, err := frontend.OpenUnknownAccentRules(a("rules", "unknown-accent.json"))
	if err != nil {
		return nil, err
	}
	english, err := frontend.OpenEnglishReader(a("rules"), codec, unknownAccent)
	if err != nil {
		return nil, err
	}
	chunker, err := cabocha.LoadCRF(a("cabocha", "standard-chunk.ipa.model"))
	if err != nil {
		return nil, fmt.Errorf("chunker: %w", err)
	}
	dependency, err := cabocha.LoadSVM(a("cabocha", "standard-dep.ipa.model"))
	if err != nil {
		return nil, fmt.Errorf("dependency parser: %w", err)
	}
	model, err := phrase.OpenBoundModel(o.Runtime, a("boundmodel"), codec)
	if err != nil {
		return nil, err
	}
	e := &Engine{predict: model}
	e.frontend = &phrase.Frontend{Analyzer: analyzer, Morphology: morphology, Reading: reading, Codec: codec,
		Bound: bound, Predictor: model, Numerals: numerals, UnknownKana: unknownKana, UnknownAccent: unknownAccent, Chunker: chunker, Dependency: dependency, English: english, Rendaku: rendaku, Cut: cut}
	akane := *e.frontend
	akane.Predictor = nil
	if akane.Rendaku, err = rendakuAssets.Rules("kansai"); err != nil {
		model.Close()
		return nil, err
	}
	if akane.Analyzer, err = frontend.OpenAnalyzer(a("dict-kansai")); err != nil {
		model.Close()
		return nil, fmt.Errorf("Kansai dictionary: %w", err)
	}
	if akane.Reading, err = openRules(a("rules", "reading-rules-kansai.txt"), frontend.ParseReadingRules); err != nil {
		model.Close()
		return nil, err
	}
	if akane.Bound, err = openRules(a("rules", "bound-rules-kansai.txt"), phrase.ParseBoundRules); err != nil {
		model.Close()
		return nil, err
	}
	if akane.Numerals, err = openRules(a("rules", "bancharcx-kansai.txt"), numeral.ParseRules); err != nil {
		model.Close()
		return nil, err
	}
	if akane.Cut, err = openRules(a("rules", "bancut-kansai.json"), phrase.ParseCutRules); err != nil {
		model.Close()
		return nil, err
	}
	if akane.Kansai, err = openRules(a("rules", "kansai-chunk-bound.json"), phrase.ParseKansaiChunkRules); err != nil {
		model.Close()
		return nil, err
	}
	e.akane = &akane
	akaneData, err := os.ReadFile(a("data", "controls-akane.json"))
	if err != nil {
		model.Close()
		return nil, err
	}
	if err = json.Unmarshal(akaneData, &e.akaneControls); err != nil {
		model.Close()
		return nil, err
	}
	if e.encoder, err = frontend.OpenMoraEncoder(a("data", "phonetics.json")); err != nil {
		model.Close()
		return nil, err
	}
	b, err := os.ReadFile(a("data", "controls-aoi.json"))
	if err == nil {
		err = json.Unmarshal(b, &e.controls)
	}
	if err != nil {
		model.Close()
		return nil, err
	}
	if e.synth, err = synthesis.OpenWithProfile(a("synth"), onnx.OriginalKernelProfile{Threads: o.OriginalThreads, AVX512Convolutions: o.OriginalAVX512Convolutions}); err != nil {
		model.Close()
		return nil, err
	}
	return e, nil
}

func openRules[T any](path string, parse func(io.Reader) (T, error)) (T, error) {
	var zero T
	f, err := os.Open(path)
	if err != nil {
		return zero, err
	}
	defer f.Close()
	return parse(f)
}

// Voices lists both supported voices, preserving Aoi as the default.
func (e *Engine) Voices() []engine.Voice {
	return []engine.Voice{{ID: VoiceAoi, Name: "aoi", SampleRate: 48000}, {ID: VoiceAkane, Name: "akane", SampleRate: 48000}}
}

// ReadingVoice returns the selected voice's formatted reading.
func (e *Engine) ReadingVoice(text, voice string) ([]string, error) {
	if voice == VoiceAkane {
		return e.akane.KanaText(text)
	}
	if voice == "" || voice == VoiceAoi {
		return e.frontend.KanaText(text)
	}
	return nil, fmt.Errorf("voice %q is not supported by the portable engine", voice)
}

// Reading returns the formatted reading of each sentence of text.
func (e *Engine) Reading(text string) ([]string, error) { return e.frontend.KanaText(text) }

// Synthesize renders text to a WAV file.
func (e *Engine) Synthesize(ctx context.Context, r engine.Request) ([]byte, error) {
	wav, _, err := e.SynthesizeTimed(ctx, r)
	return wav, err
}

// SynthesizeTimed preserves synthesis and records actual sentence PCM boundaries.
func (e *Engine) SynthesizeTimed(ctx context.Context, r engine.Request) ([]byte, []engine.Cue, error) {
	r.Defaults()
	if err := r.Validate(); err != nil {
		return nil, nil, err
	}
	if r.Voice != "" && r.Voice != VoiceAoi && r.Voice != VoiceAkane {
		return nil, nil, fmt.Errorf("voice %q is not supported by the portable engine", r.Voice)
	}
	controls := e.controls
	textFrontend := e.frontend
	if r.Voice == VoiceAkane {
		controls = e.akaneControls
		textFrontend = e.akane
	}
	// The engine feeds the acoustic model log-scale values (observed on the
	// model inputs for rate 1.5/0.5, pitch 1.2, volume 0.5 and emphasis 1.5).
	controls.Speed = float32(-math.Log(r.Rate))
	controls.Pitch = float32(math.Log(r.Pitch))
	controls.Intonation = float32(r.Emphasis)
	controls.Loudness = float32(math.Log10(r.Volume) / 5)
	kana, err := textFrontend.KanaText(r.Text)
	if err != nil {
		return nil, nil, err
	}
	var pcm []byte
	var cues []engine.Cue
	texts := phrase.SplitSentences(r.Text)
	addCue := func(index, start int) {
		if text := strings.TrimSpace(texts[index]); text != "" {
			cues = append(cues, engine.Cue{Text: text, Start: time.Duration(start/2) * time.Second / 48000, End: time.Duration(len(pcm)/2) * time.Second / 48000})
		}
	}
	for index, k := range kana {
		start := len(pcm)
		if emptyReading(k) {
			// The original emits begin/end silence for an empty sentence,
			// even when another sentence in the same request has speech.
			samples := (controls.BeginPauseMS + controls.EndPauseMS) * 48
			pcm = append(pcm, make([]byte, 2*samples)...)
			addCue(index, start)
			continue
		}
		sentence, err := e.encoder.ParseFormattedKana(k)
		if err != nil {
			return nil, nil, err
		}
		sequence, err := e.encoder.CompilePhones(sentence, 0xef)
		if err != nil {
			return nil, nil, err
		}
		input, err := synthesis.FromPhones(sequence, controls)
		if err != nil {
			return nil, nil, err
		}
		part, err := e.synth.Render(ctx, input)
		if err != nil {
			return nil, nil, err
		}
		pcm = append(pcm, part...)
		addCue(index, start)
	}
	if len(pcm) == 0 {
		// text without any word: the engine renders only the begin and end pauses
		samples := (e.controls.BeginPauseMS + e.controls.EndPauseMS) * 48
		pcm = make([]byte, 2*samples)
	}
	wav, err := engine.WAV(pcm, 48000)
	return wav, cues, err
}

// emptyReading reports a formatted reading with no moras (<S><F>).
func emptyReading(k string) bool {
	k = strings.TrimPrefix(k, "<S>")
	return k == "" || (len(k) == 3 && k[0] == '<' && k[2] == '>')
}

// Close releases the models.
func (e *Engine) Close() error {
	e.predict.Close()
	return e.synth.Close()
}
