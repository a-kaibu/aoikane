// Compare all original vocoder convolutions in one continuous pure-Go run.
// Native intermediate tensors are references only, never graph inputs.
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
}
type operator struct {
	Operation int
	Node, Op  string
	Names     []string
	Tensors   []tensor
}
type manifest struct {
	ModelHash  string `json:"model_sha256"`
	ScriptHash string `json:"torchscript_sha256"`
	DLLHash    string `json:"dll_sha256"`
	Hash       string `json:"output_sha256"`
	WaveHash   string `json:"native_wave_sha256"`
	Count      int    `json:"value_count"`
	Threads    int    `json:"original_threads"`
	Call       int    `json:"call_index"`
	Inputs     map[string]tensor
	Operators  []operator
	Wave       tensor
}
type difference struct {
	Values       int    `json:"values"`
	Different    int    `json:"bit_differences"`
	First        int    `json:"first_index"`
	GoBits       string `json:"go_bits,omitempty"`
	OriginalBits string `json:"original_bits,omitempty"`
}
type row struct {
	Operation int                   `json:"operation"`
	Node      string                `json:"node"`
	Ports     map[string]difference `json:"ports"`
}
type firstDifference struct {
	Operation  int        `json:"operation"`
	Node       string     `json:"node"`
	Port       string     `json:"port"`
	TensorName string     `json:"tensor_name"`
	Difference difference `json:"difference"`
}
type report struct {
	IntelProfile       bool             `json:"intel_profile"`
	AVX512Convolutions bool             `json:"avx512_convolutions"`
	Threads            int              `json:"original_threads"`
	Values             int              `json:"values"`
	Different          int              `json:"bit_differences"`
	Hash               string           `json:"original_value_sha256"`
	Wave               difference       `json:"wave"`
	First              *firstDifference `json:"first_difference"`
	Operators          []row            `json:"operators"`
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
		if n < 1 || n > (64<<20)/count {
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

func tensorDiff(got, want onnx.Value) (difference, error) {
	d := difference{Values: len(want.F), First: -1}
	if got.Kind != 1 || want.Kind != 1 || !slices.Equal(got.Shape, want.Shape) || len(got.F) != len(want.F) {
		return d, fmt.Errorf("tensor shape/type differs: Go=%v original=%v", got.Shape, want.Shape)
	}
	for i, v := range want.F {
		a, b := math.Float32bits(got.F[i]), math.Float32bits(v)
		if a != b {
			d.Different++
			if d.First < 0 {
				d.First = i
				d.GoBits = fmt.Sprintf("%08x", a)
				d.OriginalBits = fmt.Sprintf("%08x", b)
			}
		}
	}
	return d, nil
}
func sameTensor(a, b tensor) bool {
	return a.ID == b.ID && a.DType == b.DType && slices.Equal(a.Shape, b.Shape) && slices.Equal(a.Stride, b.Stride)
}
func readManifest(root, model, observation, originalWAV string, threads int) (manifest, error) {
	var m manifest
	raw, err := os.ReadFile(filepath.Join(root, "operators.json"))
	if err != nil {
		return m, err
	}
	if err = json.Unmarshal(raw, &m); err != nil {
		return m, err
	}
	raw, err = os.ReadFile(model)
	if err != nil {
		return m, err
	}
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != "ff245a6149817c01c2d559e49087892623116766d1f3c95e7996bac23e6887e0" || m.ModelHash != "ff245a6149817c01c2d559e49087892623116766d1f3c95e7996bac23e6887e0" || m.ScriptHash != "f63fe6b24c0de3cac10c6720adc3002166f1856493d8f91105ca77c4e6f45737" || m.DLLHash != "00ca5c937940a94d052dbe52ec5fb0909db11ffbfccfd6e65e19b468169ff5e5" || m.Threads != threads || m.Call != 1 || len(m.Operators) != 126 || len(m.Inputs) != 3 {
		return m, fmt.Errorf("original vocoder provenance/thread/count differs")
	}
	raw, err = os.ReadFile(observation)
	if err != nil {
		return m, err
	}
	var valid struct {
		Equal            bool
		Usable           bool `json:"usable_as_numerical_reference"`
		Observed         struct{ SHA256 string }
		OrdinaryOriginal struct{ SHA256 string } `json:"ordinary_original"`
	}
	if err = json.Unmarshal(raw, &valid); err != nil {
		return m, err
	}
	// Read the observed WAV again, so a stale report cannot bless changed data.
	wav, err := os.ReadFile(filepath.Join(root, "probe.wav"))
	if err != nil {
		return m, err
	}
	original, err := os.ReadFile(originalWAV)
	if err != nil {
		return m, err
	}
	if !valid.Equal || !valid.Usable || !slices.Equal(wav, original) || valid.Observed.SHA256 != fmt.Sprintf("%x", sha256.Sum256(wav)) || valid.OrdinaryOriginal.SHA256 != valid.Observed.SHA256 {
		return m, fmt.Errorf("observation was not validated against ordinary original WAV")
	}
	raw, err = os.ReadFile(filepath.Join(root, "events.json"))
	if err != nil {
		return m, err
	}
	var events []struct {
		Event, Model, Op              string
		Synthetic                     bool
		Run, Operation, Threads, Call int
		Inputs                        []tensor
		Output                        tensor
	}
	if err = json.Unmarshal(raw, &events); err != nil {
		return m, err
	}
	config, run, waveRun := 0, 0, 0
	inputs := map[int][]tensor{}
	outputs := map[int]tensor{}
	for _, e := range events {
		if !ordinaryVocoderEvent(e.Event, e.Synthetic) {
			return m, fmt.Errorf("ordinary vocoder capture must be error-free and without JIT tracking or synthetic probes")
		}
		switch e.Event {
		case "torch_operation_configuration":
			config++
			if e.Model != "vocoder" || e.Threads != threads || e.Call != 1 {
				return m, fmt.Errorf("original vocoder configuration differs")
			}
		case "torch_jit_forward":
			if run == 0 && len(e.Inputs) == 3 {
				run = e.Run
				for i, name := range []string{"f0", "bap", "spc"} {
					if !sameTensor(e.Inputs[i], m.Inputs[name]) {
						return m, fmt.Errorf("original vocoder entry %s differs", name)
					}
				}
			}
		case "torch_operation_input":
			if e.Op == "convolution" {
				inputs[e.Operation] = e.Inputs
			}
		case "torch_operation_output":
			outputs[e.Operation] = e.Output
		case "torch_jit_result":
			if e.Run == run && run != 0 {
				if !sameTensor(e.Output, m.Wave) {
					return m, fmt.Errorf("original final wave metadata differs")
				}
				waveRun = e.Run
			}
		}
	}
	if config != 1 || run == 0 || waveRun != run || len(inputs) != 126 {
		return m, fmt.Errorf("complete ordinary vocoder capture required")
	}
	seen := map[string]bool{}
	last := 0
	for _, r := range m.Operators {
		refs := inputs[r.Operation]
		if r.Operation <= last || r.Node == "" || (r.Op != "Conv" && r.Op != "ConvTranspose") || len(r.Names) != 4 || len(r.Tensors) != 4 || len(refs) != 3 || seen[r.Names[3]] {
			return m, fmt.Errorf("invalid original operator %d", r.Operation)
		}
		for i := 0; i < 3; i++ {
			if !sameTensor(refs[i], r.Tensors[i]) {
				return m, fmt.Errorf("original operator %d input metadata differs", r.Operation)
			}
		}
		if !sameTensor(outputs[r.Operation], r.Tensors[3]) {
			return m, fmt.Errorf("original operator %d output metadata differs", r.Operation)
		}
		seen[r.Names[3]] = true
		last = r.Operation
	}
	return m, nil
}

func ordinaryVocoderEvent(event string, synthetic bool) bool {
	if synthetic {
		return false
	}
	switch event {
	case "forward_error", "torch_thread_probe", "torch_control_probe", "torch_control_probe_restored",
		"torch_vocoder_f0_probe", "torch_vocoder_f0_probe_restored", "torch_operation_calls",
		"torch_vocoder_inputs_probe", "torch_vocoder_inputs_probe_restored", "torch_compressor_probe", "torch_compressor_probe_restored":
		return false
	}
	return true
}
func compare(root string, m manifest, got map[string]onnx.Value) (report, error) {
	out := report{Threads: m.Threads}
	h := sha256.New()
	add := func(v onnx.Value) {
		var b [4]byte
		for _, x := range v.F {
			binary.LittleEndian.PutUint32(b[:], math.Float32bits(x))
			h.Write(b[:])
		}
	}
	for _, r := range m.Operators {
		row := row{Operation: r.Operation, Node: r.Node, Ports: map[string]difference{}}
		for _, j := range []int{0, 3, 1, 2} {
			want, err := load(root, r.Tensors[j])
			if err != nil {
				return out, err
			}
			d, err := tensorDiff(got[r.Names[j]], want)
			if err != nil {
				return out, fmt.Errorf("operation %d %s: %w", r.Operation, r.Names[j], err)
			}
			label := []string{"input", "weight", "bias", "output"}[j]
			row.Ports[label] = d
			out.Values += d.Values
			out.Different += d.Different
			add(want)
			if d.Different != 0 && out.First == nil {
				out.First = &firstDifference{Operation: r.Operation, Node: r.Node, Port: label, TensorName: r.Names[j], Difference: d}
			}
		}
		out.Operators = append(out.Operators, row)
	}
	out.Hash = fmt.Sprintf("%x", h.Sum(nil))
	if out.Values != m.Count || out.Hash != m.Hash {
		return out, fmt.Errorf("original values differ from audited manifest")
	}
	want, err := load(root, m.Wave)
	if err != nil {
		return out, err
	}
	out.Wave, err = tensorDiff(got["wave"], want)
	if err != nil {
		return out, err
	}
	h.Reset()
	add(want)
	if fmt.Sprintf("%x", h.Sum(nil)) != m.WaveHash {
		return out, fmt.Errorf("original final wave hash differs")
	}
	return out, nil
}
func run() error {
	root := flag.String("trace", "private/vocoder-observation-local", "audited ordinary vocoder trace")
	model := flag.String("model", "assets/synth/vocoder.onnx", "original vocoder ONNX graph")
	threads := flag.Int("original-threads", 1, "observed numerical thread profile (1 or 6)")
	avx512 := flag.Bool("avx512-convolutions", false, "use reconstructed one-thread AVX512 convolution order")
	intel := flag.Bool("intel-profile", false, "use experimental one-thread Intel AVX512 unary/convolution and FMA orders; strict comparison still required")
	observation := flag.String("observation-report", "", "ordinary-original WAV validation report (required)")
	originalWAV := flag.String("ordinary-original", "", "ordinary unobserved original WAV (required)")
	output := flag.String("operator-report", "", "write per-operator comparison JSON (required)")
	flag.Parse()
	if (*threads != 1 && *threads != 6) || ((*avx512 || *intel) && *threads != 1) || *observation == "" || *output == "" || *originalWAV == "" {
		return fmt.Errorf("invalid thread profile or missing required report path")
	}
	m, err := readManifest(*root, *model, *observation, *originalWAV, *threads)
	if err != nil {
		return err
	}
	g, err := onnx.OpenGraph(*model)
	if err != nil {
		return err
	}
	inputs := map[string]onnx.Value{}
	for _, name := range []string{"f0", "bap", "spc"} {
		v, e := load(*root, m.Inputs[name])
		if e != nil {
			return e
		}
		inputs[name] = v
	}
	requested := []string{"wave"}
	seen := map[string]bool{"wave": true}
	for _, r := range m.Operators {
		for _, name := range r.Names {
			if !seen[name] {
				seen[name] = true
				requested = append(requested, name)
			}
		}
	}
	profile := onnx.OriginalKernelProfile{Threads: *threads, AVX512Convolutions: *avx512}
	if *intel {
		profile.AVX512Convolutions = true
		profile.AVX512Unary = true
		profile.FMAEmbeddings = true
		profile.FMASmallConvolutions = true
		profile.FMAProjections = true
	}
	got, err := g.RunOriginalKernelsWithProfile(inputs, requested, profile)
	if err != nil {
		return err
	}
	r, err := compare(*root, m, got)
	if err != nil {
		return err
	}
	r.AVX512Convolutions = profile.AVX512Convolutions
	r.IntelProfile = *intel
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(*output, append(raw, '\n'), 0644); err != nil {
		return err
	}
	fmt.Printf("126 continuous original vocoder operators: %d values, %d bit differences; wave %d values, %d differences; native SHA256 %s\n", r.Values, r.Different, r.Wave.Values, r.Wave.Different, r.Hash)
	if r.First != nil {
		raw, _ := json.Marshal(r.First)
		fmt.Println("First difference:", string(raw))
	}
	if r.Different != 0 || r.Wave.Different != 0 {
		return fmt.Errorf("pure-Go vocoder differs from original engine")
	}
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
