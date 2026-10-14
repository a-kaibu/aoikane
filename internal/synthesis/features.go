package synthesis

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/a-kaibu/aoikane/internal/onnx"
)

type modelTensor struct {
	Name  string
	Shape []int64
	Type  int
	Data  []byte
}

type modelOutput struct {
	Name  string
	Shape []int64
	Data  []float32
}

// acousticFeatures executes FeatNet with the reconstructed original MKL/oneDNN
// order, keeping the validated phoneme and downstream acoustic contracts.
func (e *Engine) acousticFeatures(tensors []modelTensor) ([]modelOutput, error) {
	return runOriginalModelWithProfile(e.featnet, tensors, []string{"duration", "phoneme_f0", "phoneme_power", "f0", "power", "bap", "spc"}, e.profile)
}

// runOriginalModel keeps the validated tensor contracts without importing a
// native library adapter. Only the model's real inputs enter graph execution.
func runOriginalModel(g *onnx.Graph, tensors []modelTensor, names []string, profile ...int) ([]modelOutput, error) {
	threads := 6
	if len(profile) != 0 && profile[0] != 0 {
		threads = profile[0]
	}
	return runOriginalModelWithProfile(g, tensors, names, onnx.OriginalKernelProfile{Threads: threads})
}

func runOriginalModelWithProfile(g *onnx.Graph, tensors []modelTensor, names []string, profile onnx.OriginalKernelProfile) ([]modelOutput, error) {
	// Preserve the six-thread default for internal zero-value Engine callers.
	if profile.Threads == 0 {
		profile.Threads = 6
	}
	inputs := make(map[string]onnx.Value, len(tensors))
	for _, t := range tensors {
		v := onnx.Value{Kind: t.Type, Shape: t.Shape}
		switch t.Type {
		case 1:
			if len(t.Data)%4 != 0 {
				return nil, fmt.Errorf("invalid FLOAT input %s", t.Name)
			}
			v.F = make([]float32, len(t.Data)/4)
			for i := range v.F {
				v.F[i] = math.Float32frombits(binary.LittleEndian.Uint32(t.Data[i*4:]))
			}
		case 7:
			if len(t.Data)%8 != 0 {
				return nil, fmt.Errorf("invalid INT64 input %s", t.Name)
			}
			v.I = make([]int64, len(t.Data)/8)
			for i := range v.I {
				v.I[i] = int64(binary.LittleEndian.Uint64(t.Data[i*8:]))
			}
		default:
			return nil, fmt.Errorf("unsupported model input type %d", t.Type)
		}
		inputs[t.Name] = v
	}
	values, err := g.RunOriginalKernelsWithProfile(inputs, names, profile)
	if err != nil {
		return nil, err
	}
	outputs := make([]modelOutput, len(names))
	for i, name := range names {
		v := values[name]
		if v.Kind != 1 {
			return nil, fmt.Errorf("invalid model output type for %s", name)
		}
		outputs[i] = modelOutput{Name: name, Shape: v.Shape, Data: v.F}
	}
	return outputs, nil
}
