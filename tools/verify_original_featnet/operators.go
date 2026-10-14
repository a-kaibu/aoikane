package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"

	"github.com/a-kaibu/aoikane/internal/onnx"
)

type originalOperator struct {
	Operation     int
	Node, Op      string
	InputName     string `json:"input_name"`
	OutputName    string `json:"output_name"`
	Input, Output tensor
}

type operatorManifest struct {
	ModelHash  string `json:"model_sha256"`
	DLLHash    string `json:"dll_sha256"`
	TraceHash  string `json:"trace_sha256"`
	Threads    int    `json:"original_threads"`
	Run        int
	Operators  []originalOperator
	ValueCount int    `json:"value_count"`
	ValueHash  string `json:"value_sha256"`
}

func readOperators(path, root, model string, threads int) (*operatorManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m operatorManifest
	if err = json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	graph, err := os.ReadFile(model)
	if err != nil {
		return nil, err
	}
	trace, err := os.ReadFile(filepath.Join(root, "events.json"))
	if err != nil {
		return nil, err
	}
	if m.ModelHash != "0341bdce841395eec505afab8d804038b681fe682cd1bb43894e63797bd4a142" || m.ModelHash != fmt.Sprintf("%x", sha256.Sum256(graph)) || m.TraceHash != fmt.Sprintf("%x", sha256.Sum256(trace)) || m.DLLHash != "00ca5c937940a94d052dbe52ec5fb0909db11ffbfccfd6e65e19b468169ff5e5" || m.Threads != threads || m.Run < 1 || len(m.Operators) != 118 || m.ValueCount < 1 {
		return nil, fmt.Errorf("original operator manifest provenance/count differs")
	}
	var events []struct {
		Event          string
		Run, Operation int
		Op             string
		Inputs         []tensor
		Output         tensor
	}
	if err = json.Unmarshal(trace, &events); err != nil {
		return nil, err
	}
	inputs, outputs := map[int]tensor{}, map[int]tensor{}
	nativeOps := map[int]string{}
	for _, e := range events {
		if e.Event == "forward_error" {
			return nil, fmt.Errorf("original observation contains an error")
		}
		if e.Run != m.Run {
			continue
		}
		switch e.Event {
		case "torch_operation_input":
			if len(e.Inputs) == 0 {
				return nil, fmt.Errorf("original operator input missing")
			}
			inputs[e.Operation], nativeOps[e.Operation] = e.Inputs[0], e.Op
		case "torch_operation_output":
			outputs[e.Operation] = e.Output
		}
	}
	same := func(a, b tensor) bool {
		return a.ID == b.ID && a.DType == b.DType && slices.Equal(a.Shape, b.Shape) && slices.Equal(a.Stride, b.Stride)
	}
	counts := map[string]int{}
	for i, r := range m.Operators {
		native := map[string]string{"Conv": "convolution", "MatMul": "linear", "Exp": "exp", "Log": "log"}[r.Op]
		if r.Operation != i+1 || r.Node == "" || r.InputName == "" || r.OutputName == "" || native == "" || nativeOps[r.Operation] != native || !same(inputs[r.Operation], r.Input) || !same(outputs[r.Operation], r.Output) {
			return nil, fmt.Errorf("original operator %d metadata differs from trace", i+1)
		}
		counts[r.Op]++
	}
	if counts["Conv"] != 100 || counts["MatMul"] != 13 || counts["Exp"] != 4 || counts["Log"] != 1 {
		return nil, fmt.Errorf("original operator types/counts differ")
	}
	return &m, nil
}

type tensorDifference struct {
	Values       int    `json:"values"`
	Different    int    `json:"bit_differences"`
	First        int    `json:"first_index"`
	GoBits       string `json:"go_bits,omitempty"`
	OriginalBits string `json:"original_bits,omitempty"`
}

type operatorComparison struct {
	Operation     int    `json:"operation"`
	Node          string `json:"node"`
	Input, Output tensorDifference
}

type firstOperatorDifference struct {
	Operation  int              `json:"operation"`
	Node       string           `json:"node"`
	Port       string           `json:"port"`
	TensorName string           `json:"tensor_name"`
	Difference tensorDifference `json:"difference"`
}

type operatorReport struct {
	FMAProjections       bool                     `json:"fma_projections"`
	FMASmallConvolutions bool                     `json:"fma_small_convolutions"`
	AVX512Unary          bool                     `json:"avx512_unary"`
	FMAEmbeddings        bool                     `json:"fma_embeddings"`
	AVX512Convolutions   bool                     `json:"avx512_convolutions"`
	Run                  int                      `json:"run"`
	Threads              int                      `json:"original_threads"`
	Values               int                      `json:"values"`
	Different            int                      `json:"bit_differences"`
	ValueHash            string                   `json:"original_value_sha256"`
	First                *firstOperatorDifference `json:"first_difference"`
	Operators            []operatorComparison     `json:"operators"`
}

func (m *operatorManifest) compare(root string, got map[string]onnx.Value) (*operatorReport, error) {
	report := &operatorReport{Run: m.Run, Threads: m.Threads}
	hash := sha256.New()
	for _, r := range m.Operators {
		row := operatorComparison{Operation: r.Operation, Node: r.Node}
		for port, ref := range []tensor{r.Input, r.Output} {
			name, diff, label := r.InputName, &row.Input, "input"
			if port == 1 {
				name, diff, label = r.OutputName, &row.Output, "output"
			}
			want, err := load(root, ref)
			if err != nil {
				return nil, err
			}
			v, ok := got[name]
			if !ok || v.Kind != 1 || want.Kind != 1 || !slices.Equal(v.Shape, want.Shape) || len(v.F) != len(want.F) {
				return nil, fmt.Errorf("operation %d %s shape/type differs: Go=%v original=%v", r.Operation, name, v.Shape, want.Shape)
			}
			diff.Values, diff.First = len(want.F), -1
			var bytes [4]byte
			for i, x := range want.F {
				original, actual := math.Float32bits(x), math.Float32bits(v.F[i])
				binary.LittleEndian.PutUint32(bytes[:], original)
				hash.Write(bytes[:])
				if original != actual {
					diff.Different++
					if diff.First < 0 {
						diff.First = i
						diff.GoBits = fmt.Sprintf("%08x", actual)
						diff.OriginalBits = fmt.Sprintf("%08x", original)
					}
				}
			}
			report.Values += diff.Values
			report.Different += diff.Different
			if diff.Different != 0 && report.First == nil {
				report.First = &firstOperatorDifference{Operation: r.Operation, Node: r.Node, Port: label, TensorName: name, Difference: *diff}
			}
		}
		report.Operators = append(report.Operators, row)
	}
	report.ValueHash = fmt.Sprintf("%x", hash.Sum(nil))
	if report.Values != m.ValueCount || report.ValueHash != m.ValueHash {
		return nil, fmt.Errorf("original operator values differ from audited manifest")
	}
	return report, nil
}
