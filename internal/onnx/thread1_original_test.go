package onnx

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"slices"
	"testing"
)

func TestOriginalThread1LinearCaptured(t *testing.T) {
	data, err := os.ReadFile("../../private/thread1-local-ops/linears.json")
	if os.IsNotExist(err) {
		t.Skip("private original one-thread capture unavailable")
	}
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != "a1a8768c8104d891f62fe138ae19b69761c1ee0c182e91942097d7797fd35391" {
		t.Fatal("original one-thread capture differs")
	}
	var records []struct {
		Node, Op            string
		InputShape          []int64 `json:"input_shape"`
		OutputShape         []int64 `json:"output_shape"`
		Input, Output, Bias []float32
	}
	if err := json.Unmarshal(data, &records); err != nil {
		t.Fatal(err)
	}
	if len(records) != 13 {
		t.Fatal("original linear count differs")
	}
	for _, r := range records {
		t.Run(r.Node, func(t *testing.T) {
			l, err := OpenLinear("../../assets/synth/featnet.onnx", r.Node)
			if err != nil {
				t.Fatal(err)
			}
			y, err := l.runOriginalLinear(r.Input, int(r.InputShape[1]), 1)
			if err != nil {
				t.Fatal(err)
			}
			if len(y) != len(r.Output) {
				t.Fatal("original linear output length differs")
			}
			different := 0
			for i, v := range y {
				if len(r.Bias) != 0 {
					v = float32(v + r.Bias[i%l.Out])
				}
				if math.Float32bits(v) != math.Float32bits(r.Output[i]) {
					different++
				}
			}
			if different != 0 {
				t.Fatalf("k=%d out=%d rows=%d: %d/%d values differ", l.In, l.Out, r.InputShape[1], different, len(y))
			}
		})
	}
}

func TestOriginalThread1ConvCaptured(t *testing.T) {
	data, err := os.ReadFile("../../private/thread1-local-ops/operations.json")
	if os.IsNotExist(err) {
		t.Skip("private original one-thread capture unavailable")
	}
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != "0b5ffd7492322ea5fdd6cdd752ea5d17e09acbcaff691e55227575ed85afac73" {
		t.Fatal("original one-thread capture differs")
	}
	var records []struct {
		Node, Op      string
		InputShape    []int64 `json:"input_shape"`
		OutputShape   []int64 `json:"output_shape"`
		Input, Output []float32
	}
	if err := json.Unmarshal(data, &records); err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, r := range records {
		if r.Op != "Conv" {
			continue
		}
		count++
		t.Run(r.Node, func(t *testing.T) {
			c, err := OpenConv1D("../../assets/synth/featnet.onnx", r.Node)
			if err != nil {
				t.Fatal(err)
			}
			y, err := c.runOriginal(r.Input, int(r.InputShape[2]), 1)
			if err != nil {
				t.Fatal(err)
			}
			if len(y) != len(r.Output) {
				t.Fatal("original convolution output length differs")
			}
			different := 0
			for i, v := range y {
				if math.Float32bits(v) != math.Float32bits(r.Output[i]) {
					different++
				}
			}
			if different != 0 {
				t.Fatalf("k=%d in=%d out=%d length=%d: %d/%d values differ", c.Kernel, c.In, c.Out, r.InputShape[2], different, len(y))
			}
		})
	}
	if count != 100 {
		t.Fatalf("original convolution count %d differs", count)
	}
}

// Continuous Go inference from only original model entry inputs. No original
// intermediate activations or original runtime are used by this regression.
func TestOriginalThread1FeatNet(t *testing.T) {
	data, err := os.ReadFile("testdata/thread1_featnet.json")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != "192e5a071cea4c9f98804cca9318317bb5d99eba4c5a1eec3525d4914b09971e" {
		t.Fatal("original one-thread FeatNet reference differs")
	}
	type tensor struct {
		Name      string
		Shape     []int64
		FloatBits []uint32 `json:"float_bits"`
		Integers  []int64
	}
	var ref struct {
		Threads         int
		Inputs, Outputs []tensor
		ModelHash       string `json:"model_sha256"`
	}
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.Threads != 1 || len(ref.Inputs) != 14 || len(ref.Outputs) != 7 {
		t.Fatal("original one-thread model contract differs")
	}
	model, err := os.ReadFile("../../assets/synth/featnet.onnx")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(model)) != ref.ModelHash {
		t.Fatal("original FeatNet model differs")
	}
	g, err := OpenGraph("../../assets/synth/featnet.onnx")
	if err != nil {
		t.Fatal(err)
	}
	inputs := map[string]Value{}
	for _, a := range ref.Inputs {
		v := Value{Kind: 7, Shape: a.Shape, I: a.Integers}
		if a.FloatBits != nil {
			v = Value{Kind: 1, Shape: a.Shape}
			for _, b := range a.FloatBits {
				v.F = append(v.F, math.Float32frombits(b))
			}
		}
		inputs[a.Name] = v
	}
	var outputs []string
	for _, a := range ref.Outputs {
		outputs = append(outputs, a.Name)
	}
	got, err := g.RunOriginalKernelsWithThreads(inputs, outputs, 1)
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, a := range ref.Outputs {
		v := got[a.Name]
		if v.Kind != 1 || !slices.Equal(v.Shape, a.Shape) || len(v.F) != len(a.FloatBits) {
			t.Fatal("original FeatNet output length differs")
		}
		for i, b := range a.FloatBits {
			if math.Float32bits(v.F[i]) != b {
				t.Fatalf("%s value %d: %08x != %08x", a.Name, i, math.Float32bits(v.F[i]), b)
			}
		}
		total += len(v.F)
	}
	if total != 13366 {
		t.Fatal("original FeatNet output count differs")
	}
	t.Logf("%d original one-thread FeatNet values match", total)
}

// Ordinary CI original entry inputs drive the full Go vocoder. References are
// final wave tensors; native intermediate activations never enter execution.
func TestOriginalThread1Vocoder(t *testing.T) {
	data, err := os.ReadFile("testdata/thread1_vocoder.json")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != "0a9572e5360ddbd039ab2ca5c5fda69765b025ecec01a4ec10fe1d83de355834" {
		t.Fatal("original one-thread vocoder reference differs")
	}
	type tensor struct {
		Name      string
		Shape     []int64
		FloatBits []uint32 `json:"float_bits"`
	}
	var ref struct {
		Threads   int
		ModelHash string `json:"model_sha256"`
		Cases     []struct {
			Run    int
			Inputs []tensor
			Output tensor
		}
	}
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.Threads != 1 || len(ref.Cases) != 2 {
		t.Fatal("original vocoder contract differs")
	}
	model, err := os.ReadFile("../../assets/synth/vocoder.onnx")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(model)) != ref.ModelHash {
		t.Fatal("original vocoder model differs")
	}
	g, err := OpenGraph("../../assets/synth/vocoder.onnx")
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, c := range ref.Cases {
		t.Run(fmt.Sprintf("run-%d", c.Run), func(t *testing.T) {
			if len(c.Inputs) != 3 || c.Output.Name != "wave" {
				t.Fatal("original vocoder tensor contract differs")
			}
			inputs := map[string]Value{}
			for _, a := range c.Inputs {
				v := Value{Kind: 1, Shape: a.Shape, F: make([]float32, len(a.FloatBits))}
				for i, b := range a.FloatBits {
					v.F[i] = math.Float32frombits(b)
				}
				inputs[a.Name] = v
			}
			got, err := g.RunOriginalKernelsWithThreads(inputs, []string{"wave"}, 1)
			if err != nil {
				t.Fatal(err)
			}
			v := got["wave"]
			a := c.Output
			if v.Kind != 1 || !slices.Equal(v.Shape, a.Shape) || len(v.F) != len(a.FloatBits) {
				t.Fatal("original vocoder output shape differs")
			}
			for i, b := range a.FloatBits {
				if math.Float32bits(v.F[i]) != b {
					t.Fatalf("wave value %d: %08x != %08x", i, math.Float32bits(v.F[i]), b)
				}
			}
			total += len(v.F)
		})
	}
	if total != 50160 {
		t.Fatalf("original vocoder output count %d differs", total)
	}
	t.Logf("%d original one-thread vocoder values match", total)
}
