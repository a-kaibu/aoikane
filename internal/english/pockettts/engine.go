// Package pockettts runs the Pocket TTS English inference model.
package pockettts

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/a-kaibu/aoikane/internal/english/pockettts/internal/modelcfg"
	nativemodel "github.com/a-kaibu/aoikane/internal/english/pockettts/internal/native"
	"github.com/a-kaibu/aoikane/internal/english/pockettts/internal/nativert"
	"github.com/a-kaibu/aoikane/internal/english/pockettts/internal/safetensors"
	"github.com/a-kaibu/aoikane/internal/english/pockettts/internal/text"
	"github.com/a-kaibu/aoikane/internal/english/pockettts/internal/tokenizer"
	"github.com/cwbudde/algo-tensor/ops"
	"github.com/cwbudde/algo-tensor/tensor"
)

// SampleRate is the rate of the PCM Synthesize returns.
const SampleRate = 24000

// maxTokensPerChunk is the SentencePiece token budget per synthesis chunk,
// matching the reference implementation's 50-token limit.
const maxTokensPerChunk = 50

// mimiStepsPerLatent is the Mimi decoder steps per generated latent frame.
const mimiStepsPerLatent = 16

// maxSamplerSteps bounds Options.SamplerSteps; more steps only cost time.
const maxSamplerSteps = 64

// Engine synthesizes speech with one loaded model. Its methods must not be
// called concurrently.
type Engine struct {
	model *modelcfg.ModelConfig
	tok   tokenizer.Tokenizer
	rt    *nativert.Runtime
}

// Load decodes the 6.9M REFL inference model into memory.
func Load(weights, tokenizerData []byte) (*Engine, error) {
	defaultWorkers()

	cfg, err := modelcfg.Parse(config)
	if err != nil {
		return nil, fmt.Errorf("pockettts: model config: %w", err)
	}

	tok, err := tokenizer.LoadBytes(tokenizerData, cfg.FlowLM.LookupTable.NBins)
	if err != nil {
		return nil, fmt.Errorf("pockettts: load tokenizer: %w", err)
	}

	return load(cfg, tok, weights)
}

func load(cfg *modelcfg.ModelConfig, tok tokenizer.Tokenizer, weights []byte) (*Engine, error) {
	store, err := safetensors.OpenStoreFromBytes(weights, safetensors.StoreOptions{})
	if err != nil {
		return nil, fmt.Errorf("pockettts: open weights: %w", err)
	}
	defer store.Close()

	model, err := nativemodel.LoadModelFromStore(store, nativemodel.ConfigFor(cfg))
	if err != nil {
		return nil, fmt.Errorf("pockettts: load model: %w", err)
	}

	return &Engine{model: cfg, tok: tok, rt: nativert.New(model)}, nil
}

// Close releases the model.
func (e *Engine) Close() {
	if e != nil && e.rt != nil {
		e.rt.Close()
	}
}

// Voice is a parsed voice: a voice embedding or a voice model state.
type Voice struct {
	embedding *nativert.VoiceEmbedding
	state     *safetensors.VoiceModelState
}

// ParseVoice parses a voice .safetensors file, either kind.
func (e *Engine) ParseVoice(data []byte) (*Voice, error) {
	kind, err := safetensors.InspectVoiceFileBytes(data)
	if err != nil {
		return nil, fmt.Errorf("pockettts: inspect voice: %w", err)
	}

	if kind == safetensors.VoiceFileModelState {
		state, err := safetensors.LoadVoiceModelStateFromBytes(data)
		if err != nil {
			return nil, fmt.Errorf("pockettts: load voice model state: %w", err)
		}

		if err := e.rt.CheckVoice(nil, state); err != nil {
			return nil, err
		}
		return &Voice{state: state}, nil
	}

	values, shape, err := safetensors.LoadVoiceEmbeddingFromBytes(data)
	if err != nil {
		return nil, fmt.Errorf("pockettts: load voice embedding: %w", err)
	}

	embedding := &nativert.VoiceEmbedding{Data: values, Shape: shape}
	if err := e.rt.CheckVoice(embedding, nil); err != nil {
		return nil, err
	}
	return &Voice{embedding: embedding}, nil
}

// Options controls one synthesis. Start from DefaultOptions.
type Options struct {
	// Temperature scales sampling noise; 0 is deterministic and flat.
	Temperature float64
	// EOSThreshold is the end-of-speech logit a chunk stops at; lower
	// values stop earlier.
	EOSThreshold float64
	// SamplerSteps is the flow decode steps per frame (1 to 64).
	SamplerSteps int
	// Seed makes the output reproducible for one build and platform.
	Seed uint64
}

// DefaultOptions returns the defaults of the loaded model.
func (e *Engine) DefaultOptions() Options {
	return Options{Temperature: e.model.DefaultTemperature, EOSThreshold: e.model.DefaultEOSThreshold, SamplerSteps: 1}
}

func (o Options) validate() error {
	switch {
	case math.IsNaN(o.Temperature) || math.IsInf(o.Temperature, 0) || o.Temperature < 0 || o.Temperature > 2:
		return errors.New("temperature must be in [0, 2]")
	case math.IsNaN(o.EOSThreshold) || math.IsInf(o.EOSThreshold, 0):
		return errors.New("EOS threshold must be finite")
	case o.SamplerSteps < 1 || o.SamplerSteps > maxSamplerSteps:
		return fmt.Errorf("sampler steps must be in [1, %d]", maxSamplerSteps)
	}

	return nil
}

// Synthesize calls output with the PCM for each text chunk in playback order.
func (e *Engine) Synthesize(ctx context.Context, input string, voice *Voice, opts Options, output func(string, []float32) error) error {
	err := opts.validate()
	if err != nil {
		return fmt.Errorf("pockettts: %w", err)
	}

	chunks, err := e.Prepare(input)
	if err != nil {
		return err
	}

	// The model weights are read-only and each chunk owns its generation state.
	// Bound native parallelism to two chunks so memory use stays bounded.
	parallelism := 1
	if runtime.GOMAXPROCS(0) >= 8 {
		parallelism = 2
	}
	type chunkResult struct {
		pcm []float32
		err error
	}
	left := 0
	for batch := range slices.Chunk(chunks, parallelism) {
		if err := ctx.Err(); err != nil {
			return err
		}
		var results [2]chunkResult
		run := func(index int) {
			chunk := batch[index]
			results[index].pcm, results[index].err = e.SynthesizeChunk(ctx, chunk, voice, opts, left+index)
		}
		var workers sync.WaitGroup
		if len(batch) > 1 {
			workers.Go(func() { run(1) })
		}
		run(0)
		workers.Wait()
		for i, chunk := range batch {
			if err := results[i].err; err != nil {
				return fmt.Errorf("pockettts: chunk %d of %d: %w", left+i+1, len(chunks), err)
			}
			if err := output(strings.TrimSpace(chunk.Text), results[i].pcm); err != nil {
				return err
			}
		}
		left += len(batch)
	}
	return nil
}

// Chunk holds the original tokenizer output, shared by native and browser workers.
type Chunk = text.ChunkMetadata

func (e *Engine) Prepare(input string) ([]Chunk, error) {
	normalized, err := text.Normalize(input)
	if err != nil {
		return nil, fmt.Errorf("pockettts: %w", err)
	}

	chunks, err := text.PrepareChunks(normalized, e.tok, maxTokensPerChunk, text.OptionsFor(e.model))
	if err != nil {
		return nil, fmt.Errorf("pockettts: prepare text: %w", err)
	}

	if len(chunks) == 0 {
		return nil, errors.New("pockettts: text produced no chunks")
	}

	return chunks, nil
}

func (e *Engine) SynthesizeChunk(ctx context.Context, chunk Chunk, voice *Voice, opts Options, index int) ([]float32, error) {
	if len(chunk.TokenIDs) == 0 || len(chunk.TokenIDs) > maxTokensPerChunk || index < 0 {
		return nil, errors.New("pockettts: invalid synthesis chunk")
	}
	maxSteps := text.EstimateMaxFrames(len(chunk.TokenIDs), text.DefaultMimiFrameRate)
	cfg := nativert.Config{
		Temperature:        opts.Temperature,
		EOSThreshold:       opts.EOSThreshold,
		MaxSteps:           maxSteps,
		EstimatedMaxSteps:  maxSteps,
		SamplerDecodeSteps: opts.SamplerSteps,
		FramesAfterEOS:     e.model.FramesAfterEOS(chunk.FramesAfterEOS()),
		MimiStepsPerLatent: mimiStepsPerLatent,
		MimiSequenceLength: maxSteps * mimiStepsPerLatent,
		Rand:               rand.New(rand.NewSource(int64(opts.Seed + uint64(index)))),
	}
	if voice != nil {
		cfg.VoiceEmbedding, cfg.VoiceModelState = voice.embedding, voice.state
	}
	return e.rt.GenerateAudio(ctx, chunk.TokenIDs, cfg)
}

var setWorkers sync.Once

func defaultWorkers() {
	setWorkers.Do(func() {
		workers := min(runtime.GOMAXPROCS(0), 6)
		ops.SetConvWorkers(workers)
		tensor.SetWorkers(workers)
	})
}

//go:embed english_7m_refl_2026-10.yaml
var config []byte
