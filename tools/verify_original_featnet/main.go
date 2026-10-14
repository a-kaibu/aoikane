// Compare a continuous pure-Go FeatNet run with original engine observations.
// This tool imports no inference adapter or shared-library loader.
package main

import (
	"github.com/a-kaibu/aoikane/internal/onnx"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
)

type tensor struct {
	ID, DType     int
	Shape, Stride []int64
	Tokens        map[string]tensor
	Members       []tensor
}
type event struct {
	Event  string
	Run    int
	Inputs []tensor
	Output tensor
}

func load(root string, t tensor) (onnx.Value, error) {
	v := onnx.Value{Kind: 1, Shape: t.Shape}
	size := 4
	if t.DType == 4 {
		size, v.Kind = 8, 7
	}
	if (t.DType != 4 && t.DType != 6) || len(t.Shape) != len(t.Stride) || len(t.Shape) < 1 || len(t.Shape) > 3 {
		return v, fmt.Errorf("invalid original tensor metadata")
	}
	data, err := os.ReadFile(filepath.Join(root, fmt.Sprintf("model-%d.bin", t.ID)))
	if err != nil {
		return v, err
	}
	count := int64(1)
	for _, n := range t.Shape {
		if n < 1 || n > (32<<20)/count {
			return v, fmt.Errorf("invalid original tensor shape")
		}
		count *= n
	}
	for i := int64(0); i < count; i++ {
		remain, offset := i, int64(0)
		for j := len(t.Shape) - 1; j >= 0; j-- {
			offset += (remain % t.Shape[j]) * t.Stride[j]
			remain /= t.Shape[j]
		}
		if offset < 0 || offset >= int64(len(data)/size) {
			return v, fmt.Errorf("original tensor storage truncated")
		}
		if v.Kind == 1 {
			v.F = append(v.F, math.Float32frombits(binary.LittleEndian.Uint32(data[offset*4:])))
		} else {
			v.I = append(v.I, int64(binary.LittleEndian.Uint64(data[offset*8:])))
		}
	}
	return v, nil
}

func run() error {
	root := flag.String("trace", "private/long-mora-fresh", "original trace directory")
	model := flag.String("model", "assets/synth/featnet.onnx", "audited FeatNet graph")
	contracts := flag.String("contracts", "private/model-trace/model-contracts.json", "original input/output contracts")
	onnxOrder := flag.Bool("onnx-order", false, "compare previous ONNX/MLAS order instead of original MKL/oneDNN kernels")
	threads := flag.Int("original-threads", 6, "observed original thread profile (1 or 6)")
	avx512 := flag.Bool("avx512-convolutions", false, "use reconstructed one-thread AVX512 convolution order")
	unary := flag.Bool("avx512-unary", false, "use reconstructed Intel VML unary orders")
	fmaEmbedding := flag.Bool("fma-embeddings", false, "use reconstructed one-thread Intel FMA embeddings")
	pointwise := flag.Bool("fma-small-convolutions", false, "use reconstructed Intel serial and blocked im2col FMA orders")
	flag.BoolVar(pointwise, "fma-pointwise-convolutions", false, "deprecated alias for -fma-small-convolutions")
	projections := flag.Bool("fma-projections", false, "use reconstructed Intel serial and AVX512 FMA linear projections")
	operators := flag.String("operators", "", "audited original operator manifest for continuous layer comparisons")
	reportPath := flag.String("operator-report", "", "write original layer comparison JSON")
	flag.Parse()
	if (*threads != 1 && *threads != 6) || (*avx512 && (*threads != 1 || *onnxOrder)) || (*reportPath != "" && *operators == "") || (*unary && *onnxOrder) || (*fmaEmbedding && (*threads != 1 || *onnxOrder)) || (*pointwise && (*threads != 1 || *onnxOrder)) || (*projections && (*threads != 1 || *onnxOrder)) {
		return fmt.Errorf("invalid original thread profile or missing operator manifest")
	}
	var manifest *operatorManifest
	var err error
	if *operators != "" {
		if *onnxOrder {
			return fmt.Errorf("operator comparison requires original numerical kernels")
		}
		manifest, err = readOperators(*operators, *root, *model, *threads)
		if err != nil {
			return err
		}
	}
	data, err := os.ReadFile(filepath.Join(*root, "events.json"))
	if err != nil {
		return err
	}
	var rawEvents []json.RawMessage
	if err = json.Unmarshal(data, &rawEvents); err != nil {
		return err
	}
	var events []event
	for _, raw := range rawEvents {
		var header struct{ Event string }
		if err := json.Unmarshal(raw, &header); err != nil {
			return err
		}
		if header.Event != "torch_jit_forward" && header.Event != "torch_jit_result" {
			continue
		}
		var e event
		if err := json.Unmarshal(raw, &e); err != nil {
			return err
		}
		events = append(events, e)
	}
	data, err = os.ReadFile(*contracts)
	if err != nil {
		return err
	}
	var contract map[string]struct{ Inputs, Outputs []string }
	if err = json.Unmarshal(data, &contract); err != nil {
		return err
	}
	c := contract["featnet"]
	if len(c.Inputs) != 14 || len(c.Outputs) != 7 {
		return fmt.Errorf("unexpected FeatNet contract")
	}
	g, err := onnx.OpenGraph(*model)
	if err != nil {
		return err
	}
	results := map[int]tensor{}
	for _, e := range events {
		if e.Event == "torch_jit_result" {
			results[e.Run] = e.Output
		}
	}
	total, bad, calls := 0, 0, 0
	matchedOperators := false
	hash := sha256.New()
	for _, e := range events {
		if e.Event != "torch_jit_forward" || len(e.Inputs) != 10 {
			continue
		}
		named := map[string]tensor{}
		for k, v := range e.Inputs[0].Tokens {
			named[k] = v
		}
		for i, k := range []string{"spk", "sty", "speed", "pitch", "into", "loud", "dur", "f0c", "pwr"} {
			named[k] = e.Inputs[i+1]
		}
		inputs := map[string]onnx.Value{}
		for _, k := range c.Inputs {
			t, ok := named[k]
			if !ok {
				return fmt.Errorf("missing original input %s", k)
			}
			v, err := load(*root, t)
			if err != nil {
				return err
			}
			inputs[k] = v
		}
		requested := slices.Clone(c.Outputs)
		if manifest != nil && manifest.Run == e.Run {
			for _, r := range manifest.Operators {
				requested = append(requested, r.InputName, r.OutputName)
			}
		}
		var got map[string]onnx.Value
		if *onnxOrder {
			got, err = g.Run(inputs, requested)
		} else {
			got, err = g.RunOriginalKernelsWithProfile(inputs, requested, onnx.OriginalKernelProfile{Threads: *threads, AVX512Convolutions: *avx512, AVX512Unary: *unary, FMAEmbeddings: *fmaEmbedding, FMASmallConvolutions: *pointwise, FMAProjections: *projections})
		}
		if err != nil {
			return err
		}
		if manifest != nil && manifest.Run == e.Run {
			matchedOperators = true
			report, err := manifest.compare(*root, got)
			if err != nil {
				return err
			}
			report.AVX512Convolutions = *avx512
			report.AVX512Unary = *unary
			report.FMAEmbeddings = *fmaEmbedding
			report.FMASmallConvolutions = *pointwise
			report.FMAProjections = *projections
			fmt.Printf("Original operators: %d values, %d bit differences, SHA256 %s\n", report.Values, report.Different, report.ValueHash)
			if report.First != nil {
				data, _ := json.Marshal(report.First)
				fmt.Printf("First original operator difference: %s\n", data)
			}
			if *reportPath != "" {
				data, err := json.MarshalIndent(report, "", "  ")
				if err != nil {
					return err
				}
				if err = os.MkdirAll(filepath.Dir(*reportPath), 0755); err != nil {
					return err
				}
				if err = os.WriteFile(*reportPath, append(data, '\n'), 0644); err != nil {
					return err
				}
			}
			total += report.Values
			bad += report.Different
		}
		out, ok := results[e.Run]
		if !ok || len(out.Members) != len(c.Outputs) {
			return fmt.Errorf("missing original FeatNet result")
		}
		for i, k := range c.Outputs {
			want, err := load(*root, out.Members[i])
			if err != nil {
				return err
			}
			v := got[k]
			if want.Kind != 1 || v.Kind != 1 || !slices.Equal(v.Shape, want.Shape) || len(v.F) != len(want.F) {
				return fmt.Errorf("%s shape/type differs", k)
			}
			diffs, maxError := 0, float64(0)
			for j, x := range v.F {
				bits := math.Float32bits(want.F[j])
				hash.Write(binary.LittleEndian.AppendUint32(nil, bits))
				if math.Float32bits(x) != bits {
					diffs++
				}
				d := math.Abs(float64(x) - float64(want.F[j]))
				if math.IsNaN(d) {
					maxError = math.Inf(1)
				} else {
					maxError = max(maxError, d)
				}
			}
			fmt.Printf("run=%d output=%s values=%d bit_differences=%d max_error=%.9g\n", e.Run, k, len(v.F), diffs, maxError)
			total += len(v.F)
			bad += diffs
		}
		calls++
	}
	if manifest != nil && !matchedOperators {
		return fmt.Errorf("original operator model entry not compared")
	}
	if total == 0 {
		return fmt.Errorf("no original FeatNet outputs compared")
	}
	fmt.Printf("Compared %d calls, %d original engine values; %d bit differences; final-output SHA256 %x\n", calls, total, bad, hash.Sum(nil))
	if bad != 0 {
		return fmt.Errorf("pure-Go FeatNet differs from original engine")
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
