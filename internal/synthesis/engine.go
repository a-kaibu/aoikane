// Package synthesis implements the portable neural audio pipeline. Japanese
// text analysis supplies Input; this package never invokes the original engine.
package synthesis

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"

	"github.com/a-kaibu/aoikane/internal/dsp"
	"github.com/a-kaibu/aoikane/internal/onnx"
)

// Input contains phoneme-level features and controls in channel-major order.
// Acoustic features and recorded waveforms are deliberately absent.
type Input struct {
	Phoneme          []int64   `json:"phn"`
	Art              []float32 `json:"art"`
	Accent           []int64   `json:"acc"`
	Punctuation      []int64   `json:"punc"`
	MoraPosition     []float32 `json:"nima"`
	PhrasePosition   []float32 `json:"niab"`
	Style            []float32 `json:"sty"`
	Speed            []float32 `json:"speed"`
	Pitch            []float32 `json:"pitch"`
	Intonation       []float32 `json:"into"`
	Loudness         []float32 `json:"loud"`
	Duration         []float32 `json:"dur"`
	F0               []float32 `json:"f0c"`
	Power            []float32 `json:"pwr"`
	Boundaries       []int     `json:"boundaries"`
	RequestedPauseMS []int     `json:"requested_pause_ms"`
}

type Engine struct {
	mu         sync.Mutex
	profile    onnx.OriginalKernelProfile
	featnet    *onnx.Graph
	vocoder    *onnx.Graph
	compressor dsp.Compressor
}

// Open loads the pure-Go models. The first argument is retained for callers
// of the former ONNX Runtime adapter and is no longer used.
func Open(_ string, modelDir string) (*Engine, error) {
	return OpenWithThreads(modelDir, 6)
}

// OpenWithThreads chooses the original model's numerical partition profile.
// It does not change Go scheduling. The audited profiles are one and six.
func OpenWithThreads(modelDir string, threads int) (*Engine, error) {
	return OpenWithProfile(modelDir, onnx.OriginalKernelProfile{Threads: threads})
}

// OpenWithProfile selects a reconstructed original numerical profile.
func OpenWithProfile(modelDir string, profile onnx.OriginalKernelProfile) (*Engine, error) {
	threads := profile.Threads
	if threads != 1 && threads != 6 {
		return nil, fmt.Errorf("original thread profile %d is not reconstructed", threads)
	}
	if (profile.AVX512Convolutions || profile.FMAEmbeddings || profile.FMASmallConvolutions || profile.FMAProjections) && threads != 1 {
		return nil, fmt.Errorf("AVX512 convolution/FMA profile requires one original thread")
	}
	b, err := os.ReadFile(filepath.Join(modelDir, "compressor.json"))
	if err != nil {
		return nil, err
	}
	e := &Engine{profile: profile}
	if err := json.Unmarshal(b, &e.compressor); err != nil {
		return nil, err
	}
	e.featnet, err = onnx.OpenGraph(filepath.Join(modelDir, "featnet.onnx"))
	if err != nil {
		return nil, err
	}
	e.vocoder, err = onnx.OpenGraph(filepath.Join(modelDir, "vocoder.onnx"))
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.vocoder != nil {
		e.vocoder = nil
	}
	if e.featnet != nil {
		e.featnet = nil
	}
	return nil
}

func floatTensor(name string, shape []int64, values []float32) (modelTensor, error) {
	b := make([]byte, len(values)*4)
	for i, v := range values {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return modelTensor{}, fmt.Errorf("%s contains non-finite values", name)
		}
		binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(v))
	}
	return modelTensor{Name: name, Type: 1, Shape: shape, Data: b}, nil
}

func intTensor(name string, shape []int64, values []int64) modelTensor {
	b := make([]byte, len(values)*8)
	for i, v := range values {
		binary.LittleEndian.PutUint64(b[i*8:], uint64(v))
	}
	return modelTensor{Name: name, Type: 7, Shape: shape, Data: b}
}

func (input Input) tensors() ([]modelTensor, error) {
	n := len(input.Phoneme)
	if n < 2 || n > 8192 || len(input.Boundaries) < 2 || len(input.RequestedPauseMS) != n {
		return nil, fmt.Errorf("invalid phoneme input length")
	}
	if len(input.Accent) != n || len(input.Punctuation) != n {
		return nil, fmt.Errorf("integer feature lengths differ")
	}
	for _, v := range input.Phoneme {
		if v < 0 || v > 51 {
			return nil, fmt.Errorf("invalid phoneme ID")
		}
	}
	for _, v := range input.Accent {
		if v < 0 || v > 3 {
			return nil, fmt.Errorf("invalid accent state")
		}
	}
	for _, v := range input.Punctuation {
		if v < 0 || v > 10 {
			return nil, fmt.Errorf("invalid punctuation state")
		}
	}
	for i, b := range input.Boundaries {
		if b < 0 || b >= n || (i > 0 && b <= input.Boundaries[i-1]) || input.Phoneme[b] != 0 {
			return nil, fmt.Errorf("invalid pause boundaries")
		}
	}
	if input.Boundaries[0] != 0 || input.Boundaries[len(input.Boundaries)-1] != n-1 {
		return nil, fmt.Errorf("missing sentence boundary padding")
	}
	for _, p := range input.RequestedPauseMS {
		if p < -1 || p > 600000 {
			return nil, fmt.Errorf("invalid requested pause")
		}
	}
	shape := []int64{1, int64(n)}
	tensors := []modelTensor{intTensor("phn", shape, input.Phoneme), intTensor("acc", shape, input.Accent), intTensor("punc", shape, input.Punctuation)}
	for _, v := range []struct {
		name     string
		channels int
		values   []float32
	}{
		{"art", 32, input.Art}, {"nima", 0, input.MoraPosition}, {"niab", 0, input.PhrasePosition}, {"sty", 8, input.Style},
		{"speed", 1, input.Speed}, {"pitch", 1, input.Pitch}, {"into", 1, input.Intonation}, {"loud", 1, input.Loudness},
		{"dur", 1, input.Duration}, {"f0c", 1, input.F0}, {"pwr", 1, input.Power},
	} {
		channels := max(v.channels, 1)
		if len(v.values) != n*channels {
			return nil, fmt.Errorf("%s feature length differs", v.name)
		}
		shape := []int64{1, int64(n)}
		if v.channels > 0 {
			shape = []int64{1, int64(channels), int64(n)}
		}
		t, err := floatTensor(v.name, shape, v.values)
		if err != nil {
			return nil, err
		}
		tensors = append(tensors, t)
	}
	return tensors, nil
}

func frameSlice(o modelOutput, start, end, channels int) ([]float32, error) {
	if len(o.Shape) != 3 || o.Shape[0] != 1 || o.Shape[1] != int64(channels) || o.Shape[2] <= 0 ||
		start < 0 || end < start || int64(end) > o.Shape[2] || len(o.Data) != channels*int(o.Shape[2]) {
		return nil, fmt.Errorf("invalid %s acoustic output shape", o.Name)
	}
	length := end - start
	out := make([]float32, channels*length)
	frames := int(o.Shape[2])
	for c := 0; c < channels; c++ {
		copy(out[c*length:(c+1)*length], o.Data[c*frames+start:c*frames+end])
	}
	return out, nil
}

// Render synthesizes PCM16 at 48 kHz from independently supplied phoneme
// features. Both FeatNet and the vocoder use the original kernels in pure Go.
func (e *Engine) Render(ctx context.Context, input Input) ([]byte, error) {
	tensors, err := input.tensors()
	if err != nil {
		return nil, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.featnet == nil || e.vocoder == nil {
		return nil, fmt.Errorf("portable synthesis engine is closed")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	acoustic, err := e.acousticFeatures(tensors)
	if err != nil {
		return nil, err
	}
	if len(acoustic) != 7 || len(acoustic[3].Shape) != 3 {
		return nil, fmt.Errorf("invalid acoustic model result")
	}
	segments, err := dsp.ScheduleSpeech(acoustic[0].Data, input.Boundaries, input.RequestedPauseMS, int(acoustic[3].Shape[2]))
	if err != nil {
		return nil, err
	}
	var pcm []byte
	for _, segment := range segments {
		pcm = append(pcm, make([]byte, segment.PrefixSilence*2)...)
		var tail []float32
		carry := 0
		for _, chunk := range segment.Chunks {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			var inputs []modelTensor
			for _, p := range []struct {
				index, channels int
				name            string
			}{{3, 1, "f0"}, {5, 5, "bap"}, {6, 60, "spc"}} {
				v, err := frameSlice(acoustic[p.index], chunk.Start, chunk.End, p.channels)
				if err != nil {
					return nil, err
				}
				t, err := floatTensor(p.name, []int64{1, int64(p.channels), int64(chunk.End - chunk.Start)}, v)
				if err != nil {
					return nil, err
				}
				inputs = append(inputs, t)
			}
			wave, err := runOriginalModelWithProfile(e.vocoder, inputs, []string{"wave"}, e.profile)
			if err != nil {
				return nil, err
			}
			if len(wave) != 1 || len(wave[0].Data) != (chunk.End-chunk.Start)*240 {
				return nil, fmt.Errorf("invalid vocoder waveform length")
			}
			power, err := frameSlice(acoustic[4], chunk.Start, chunk.End, 1)
			if err != nil {
				return nil, err
			}
			process := e.compressor.Process
			if e.profile.AVX512Unary {
				process = e.compressor.ProcessWithAVX512Unary
			}
			processed, err := process(wave[0].Data, power)
			if err != nil {
				return nil, err
			}
			chunk.Options.CarriedSkip = carry
			joined, err := dsp.JoinChunk(processed, tail, chunk.Options)
			if err != nil {
				return nil, err
			}
			tail, carry = joined.Tail, joined.CarriedSkip
			for _, v := range joined.PCM {
				pcm = binary.LittleEndian.AppendUint16(pcm, uint16(v))
			}
		}
		pcm = append(pcm, make([]byte, segment.SuffixSilence*2)...)
		if len(pcm) > 256*1024*1024 {
			return nil, fmt.Errorf("synthesized audio exceeds 256 MiB")
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return pcm, nil
}
