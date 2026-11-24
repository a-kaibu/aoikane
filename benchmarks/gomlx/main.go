// Command aoikane-gomlx-probe checks whether Aoikane ONNX graphs run on GoMLX's
// Pure Go backend. It supplies short, synthetic inputs for graph compatibility.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/gomlx/compute"
	"github.com/gomlx/compute/dtypes"
	"github.com/gomlx/compute/gobackend"
	"github.com/gomlx/compute/shapes"
	"github.com/gomlx/gomlx/core/graph"
	"github.com/gomlx/gomlx/core/tensors"
	"github.com/gomlx/gomlx/ml/model"
	"github.com/gomlx/onnx-gomlx/onnx/parser"
)

type result struct {
	Model   string            `json:"model"`
	Inputs  map[string]string `json:"inputs,omitempty"`
	Outputs []string          `json:"outputs,omitempty"`
	Stage   string            `json:"stage"`
	FirstMS float64           `json:"firstMS,omitempty"`
	WarmMS  []float64         `json:"warmMS,omitempty"`
	Error   string            `json:"error,omitempty"`
}

func main() {
	frames := flag.Int("frames", 8, "use this size for each dynamic input axis")
	fixturePath := flag.String("vocoder-fixture", "", "optional captured vocoder input JSON")
	flag.Parse()
	if *frames < 1 || *frames > 110 || flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: gomlx-probe [-frames 8] MODEL.onnx [...]")
		os.Exit(2)
	}

	var fixture *vocoderFixture
	if *fixturePath != "" {
		if flag.NArg() != 1 {
			fmt.Fprintln(os.Stderr, "--vocoder-fixture requires one model path")
			os.Exit(2)
		}
		var err error
		fixture, err = loadVocoderFixture(*fixturePath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		*frames = fixture.Frames
	}

	backend, err := gobackend.New("")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	defer backend.Finalize()

	encoder := json.NewEncoder(os.Stdout)
	failed := false
	for _, path := range flag.Args() {
		got := probe(backend, path, *frames, fixture)
		if err := encoder.Encode(got); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		failed = failed || got.Error != ""
	}
	if failed {
		os.Exit(1)
	}
}

func probe(backend compute.Backend, path string, frames int, fixture *vocoderFixture) (got result) {
	got.Model = filepath.Base(path)
	got.Stage = "parse"
	defer func() {
		if value := recover(); value != nil {
			got.Error = firstLine(fmt.Sprint(value))
		}
	}()

	onnx, err := parser.ParseFile(path)
	if err != nil {
		got.Error = firstLine(err.Error())
		return got
	}
	defer onnx.Close()

	got.Stage = "weights"
	store := model.NewStore()
	defer store.Finalize()
	if err := onnx.VariablesToScope(store.RootScope()); err != nil {
		got.Error = firstLine(err.Error())
		return got
	}

	got.Stage = "inputs"
	names, inputShapes := onnx.Inputs()
	got.Inputs = make(map[string]string, len(names))
	inputs := make([]any, len(names))
	for i, name := range names {
		input, err := probeInput(name, inputShapes[i], frames, fixture)
		if err != nil {
			got.Error = err.Error()
			return got
		}
		got.Inputs[name] = input.Shape().String()
		inputs[i] = input
		defer input.FinalizeAll()
	}

	got.Stage = "convert"
	exec, err := model.NewExec(backend, store, func(scope *model.Scope, nodes []*graph.Node) []*graph.Node {
		params := make(map[string]*graph.Node, len(names))
		for i, name := range names {
			params[name] = nodes[i]
		}
		outputs := onnx.CallGraph(scope, nodes[0].Graph(), params)
		got.Stage = "execute"
		return outputs
	})
	if err != nil {
		got.Error = firstLine(err.Error())
		return got
	}
	defer exec.Finalize()

	started := time.Now()
	outputs, err := exec.Call(inputs...)
	if fixture != nil {
		got.FirstMS = float64(time.Since(started)) / float64(time.Millisecond)
	}
	if err != nil {
		got.Error = firstLine(err.Error())
		return got
	}
	for _, output := range outputs {
		defer output.FinalizeAll()
		got.Outputs = append(got.Outputs, output.Shape().String())
		if output.Shape().DType == dtypes.Float32 {
			err := tensors.ConstFlatData(output, func(values []float32) {
				for _, value := range values {
					if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
						got.Error = "output contains non-finite samples"
						break
					}
				}
			})
			if err != nil {
				got.Error = firstLine(err.Error())
				return got
			}
		}
	}
	if got.Error == "" && fixture != nil {
		got.Stage = "audio"
		if err := fixture.save(outputs); err != nil {
			got.Error = firstLine(err.Error())
			return got
		}
		for range 3 {
			started = time.Now()
			repeat, err := exec.Call(inputs...)
			elapsed := time.Since(started)
			for _, output := range repeat {
				output.FinalizeAll()
			}
			if err != nil {
				got.Error = firstLine(err.Error())
				return got
			}
			got.WarmMS = append(got.WarmMS, float64(elapsed)/float64(time.Millisecond))
		}
	}
	if got.Error == "" {
		got.Stage = "ok"
	}
	return got
}

func firstLine(message string) string {
	line, _, _ := strings.Cut(message, "\n")
	return line
}

func probeInput(name string, shape shapes.Shape, frames int, fixture *vocoderFixture) (*tensors.Tensor, error) {
	dims := slices.Clone(shape.Dimensions)
	count := 1
	for i, dim := range dims {
		if dim == shapes.DynamicDim {
			dim = frames
			dims[i] = dim
		}
		if dim < 0 || (dim > 0 && count > (1<<20)/dim) {
			return nil, fmt.Errorf("%s: unsupported input shape %v", name, shape)
		}
		count *= dim
	}

	switch shape.DType {
	case dtypes.Int64:
		return tensors.FromFlatDataAndDimensions(make([]int64, count), dims...), nil
	case dtypes.Int32:
		return tensors.FromFlatDataAndDimensions(make([]int32, count), dims...), nil
	case dtypes.Float32:
		if fixture != nil {
			values := fixture.Inputs[name]
			if len(values) != count {
				return nil, fmt.Errorf("%s: fixture has %d values, want %d", name, len(values), count)
			}
			return tensors.FromFlatDataAndDimensions(values, dims...), nil
		}
		values := make([]float32, count)
		switch name {
		case "speed", "pitch", "into", "loud", "dur", "f0c", "pwr":
			for i := range values {
				values[i] = 1
			}
		case "f0":
			for i := range values {
				values[i] = 180
			}
		}
		return tensors.FromFlatDataAndDimensions(values, dims...), nil
	default:
		return nil, fmt.Errorf("%s: unsupported input type %s", name, shape.DType)
	}
}
