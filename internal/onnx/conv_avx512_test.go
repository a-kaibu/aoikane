package onnx

import (
	"encoding/json"
	"math"
	"os"
	"testing"
)

func TestOriginalAVX512DirectSamples(t *testing.T) {
	var ref struct {
		Operation int
		Samples   []struct {
			Channel, Frame int
			Input, Weight  []uint32
			Bias, Output   uint32
		}
	}
	raw, err := os.ReadFile("testdata/dnnl_avx512_direct.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.Operation != 72 || len(ref.Samples) != 3 {
		t.Fatal("invalid native reference")
	}
	for _, s := range ref.Samples {
		if len(s.Input) != 256*3 || len(s.Weight) != 256*3 {
			t.Fatal("invalid native sample extent")
		}
		// Extracted windows are logical input coordinates of the original
		// 256 -> 512, kernel-three convolution. Repeat the selected output
		// lane to retain valid blocked geometry without storing every weight.
		c := Conv1D{In: 256, Out: 16, Kernel: 3, Stride: 1, Dilation: 1, weight: make([]float32, 16*256*3), bias: make([]float32, 16)}
		x := make([]float32, len(s.Input))
		for i, bits := range s.Input {
			x[i] = math.Float32frombits(bits)
		}
		for oc := 0; oc < 16; oc++ {
			c.bias[oc] = math.Float32frombits(s.Bias)
			for i, bits := range s.Weight {
				c.weight[oc*256*3+i] = math.Float32frombits(bits)
			}
		}
		y, err := c.runOriginalDNNLAVX512(x, 3, 1)
		if err != nil {
			t.Fatal(err)
		}
		for lane, v := range y {
			if got := math.Float32bits(v); got != s.Output {
				t.Fatalf("native channel=%d frame=%d lane=%d: got %08x want %08x", s.Channel, s.Frame, lane, got, s.Output)
			}
		}
	}
}

func TestOriginalAVX512DirectFullCapture(t *testing.T) {
	root := "../../private/cpu-layer-ci-37920590780/cpu-original-replica-1/trace"
	raw, err := os.ReadFile(root + "/featnet-operators.json")
	if os.IsNotExist(err) {
		t.Skip("private original CPU capture not installed")
	}
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		Operators []struct {
			Operation     int
			Node          string
			Input, Output nativeVocoderTensor
		}
	}
	if err := json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if len(ref.Operators) != 118 || ref.Operators[71].Operation != 72 {
		t.Fatal("invalid native operation manifest")
	}
	r := ref.Operators[71]
	c, err := OpenConv1D("../../assets/synth/featnet.onnx", r.Node)
	if err != nil {
		t.Fatal(err)
	}
	x, want := readNativeVocoderTensor(t, root, r.Input), readNativeVocoderTensor(t, root, r.Output)
	y, err := c.runOriginalDNNLAVX512(x.F, int(x.Shape[2]), int(want.Shape[2]))
	if err != nil {
		t.Fatal(err)
	}
	if len(y) != len(want.F) {
		t.Fatal("native output extent differs")
	}
	for i, v := range y {
		if math.Float32bits(v) != math.Float32bits(want.F[i]) {
			t.Fatalf("native output %d: got %08x want %08x", i, math.Float32bits(v), math.Float32bits(want.F[i]))
		}
	}
	t.Logf("all %d original output values match", len(y))
}

func TestOriginalAVX512SmallChannels(t *testing.T) {
	testOriginalAVX512Window(t, "testdata/dnnl_avx512_small.json", 2, 8, 7, 11)
}
func TestOriginalAVX512KernelEight(t *testing.T) {
	testOriginalAVX512Window(t, "testdata/dnnl_avx512_stride.json", 64, 16, 8, 15)
}

// Windows are extracted using the native stride and padding; lane reduction
// then sees the same scalar inputs as the original selected output position.
func testOriginalAVX512Window(t *testing.T, path string, in, out, kernel, operation int) {
	var ref struct {
		Operation int
		Samples   []struct {
			Channel, Frame int
			Input, Weight  []uint32
			Bias, Output   uint32
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.Operation != operation || len(ref.Samples) != 3 {
		t.Fatal("invalid original small-channel reference")
	}
	for _, s := range ref.Samples {
		if len(s.Input) != in*kernel || len(s.Weight) != in*kernel {
			t.Fatal("invalid original sample extent")
		}
		c := Conv1D{In: in, Out: out, Kernel: kernel, weight: make([]float32, out*in*kernel), bias: make([]float32, out)}
		x := make([]float32, in*kernel)
		for i, v := range s.Input {
			x[i] = math.Float32frombits(v)
		}
		for oc := 0; oc < out; oc++ {
			c.bias[oc] = math.Float32frombits(s.Bias)
			for i, v := range s.Weight {
				c.weight[oc*in*kernel+i] = math.Float32frombits(v)
			}
		}
		y, err := c.runOriginalDNNLAVX512(x, kernel, 1)
		if err != nil {
			t.Fatal(err)
		}
		for lane, v := range y {
			if math.Float32bits(v) != s.Output {
				t.Fatalf("channel=%d frame=%d lane=%d: got %08x want %08x", s.Channel, s.Frame, lane, math.Float32bits(v), s.Output)
			}
		}
	}
}

func TestOriginalAVX512VocoderDirectCapture(t *testing.T) {
	root := "../../private/cpu-numeric-ci-37927408109/cpu-original-replica-1/vocoder-trace"
	raw, err := os.ReadFile(root + "/operators.json")
	if os.IsNotExist(err) {
		t.Skip("private original CPU capture not installed")
	}
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		Operators []struct {
			Operation int
			Node      string
			Tensors   []nativeVocoderTensor
		}
	}
	if err := json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	count := 0
	g, err := OpenGraph("../../assets/synth/vocoder.onnx")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range ref.Operators {
		if r.Operation < 11 || r.Operation > 15 {
			continue
		}
		t.Run(r.Node, func(t *testing.T) {
			var node *graphNode
			for i := range g.nodes {
				if g.nodes[i].name == r.Node {
					node = &g.nodes[i]
					break
				}
			}
			if node == nil || node.op != "Conv" || len(node.inputs) != 3 {
				t.Fatal("original Conv node missing")
			}
			constants, err := g.Run(nil, node.inputs[1:])
			if err != nil {
				t.Fatal(err)
			}
			w, b := constants[node.inputs[1]], constants[node.inputs[2]]
			c := &Conv1D{In: int(w.Shape[1]), Out: int(w.Shape[0]), Kernel: int(w.Shape[2]), weight: w.F, bias: b.F}
			if err := configureConv(c, node.attrs); err != nil {
				t.Fatal(err)
			}
			x, want := readNativeVocoderTensor(t, root, r.Tensors[0]), readNativeVocoderTensor(t, root, r.Tensors[3])
			y, err := c.runOriginalDNNLAVX512(x.F, int(x.Shape[2]), int(want.Shape[2]))
			if err != nil {
				t.Fatal(err)
			}
			if len(y) != len(want.F) {
				t.Fatal("original output extent differs")
			}
			for i, v := range y {
				if math.Float32bits(v) != math.Float32bits(want.F[i]) {
					t.Fatalf("original output %d: got %08x want %08x", i, math.Float32bits(v), math.Float32bits(want.F[i]))
				}
			}
			t.Logf("all %d original output values match", len(y))
		})
		count++
	}
	if count != 5 {
		t.Fatal("complete original direct convolution capture missing")
	}
}
