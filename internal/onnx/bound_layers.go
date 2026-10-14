package onnx

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"os"
)

// BoundLayers executes the bundled bou classifier and exposes its two
// bidirectional LSTM layers. Only the audited model is accepted; this is not
// a general ONNX graph interpreter.
type BoundLayers struct {
	layers      [2]lstmLayer
	dense, bias []float32
}

type lstmLayer struct {
	width, hidden int
	w, r, bias    []float32
}

func OpenBoundLayers(path string) (*BoundLayers, error) {
	return loadLayers(path, "14bd68227e3c8ca1d9cc933792807c6a2d95f6f8e4598288602d941a4b3abcee", []lstmSpec{
		{233, "onnx::LSTM_353", "onnx::LSTM_354", "onnx::LSTM_352"},
		{128, "onnx::LSTM_396", "onnx::LSTM_397", "onnx::LSTM_395"},
	}, "onnx::MatMul_398")
}

type lstmSpec struct {
	width   int
	w, r, b string
}

func loadLayers(path, digest string, specs []lstmSpec, dense string) (*BoundLayers, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != digest {
		return nil, fmt.Errorf("unsupported BOUND model: SHA256 differs from audited graph")
	}
	model, err := fields(data)
	if err != nil {
		return nil, err
	}
	graphData, err := oneBytes(model, 7)
	if err != nil {
		return nil, err
	}
	graph, err := fields(graphData)
	if err != nil {
		return nil, err
	}
	initializers, err := bytesFields(graph, 5)
	if err != nil {
		return nil, err
	}
	weights := make(map[string][]float32)
	for _, raw := range initializers {
		tensor, err := fields(raw)
		if err != nil {
			return nil, err
		}
		name, err := oneBytes(tensor, 8)
		if err != nil {
			return nil, err
		}
		values, err := oneBytes(tensor, 9)
		if err != nil {
			return nil, err
		}
		if len(values)%4 != 0 {
			return nil, fmt.Errorf("unaligned initializer %s", name)
		}
		v := make([]float32, len(values)/4)
		for i := range v {
			v[i] = math.Float32frombits(binary.LittleEndian.Uint32(values[4*i:]))
		}
		weights[string(name)] = v
	}
	b := &BoundLayers{}
	b.dense, b.bias = weights[dense], weights["fco.bias"]
	if len(b.dense) != 128*3 || len(b.bias) != 3 {
		return nil, fmt.Errorf("invalid BOUND classifier weights")
	}
	for i, spec := range specs {
		l := lstmLayer{width: spec.width, hidden: 64, w: weights[spec.w], r: weights[spec.r]}
		rawBias := weights[spec.b]
		if len(l.w) != 2*256*l.width || len(l.r) != 2*256*64 || len(rawBias) != 2*512 {
			return nil, fmt.Errorf("invalid LSTM %d weights", i)
		}
		l.bias = make([]float32, 2*256)
		for d := 0; d < 2; d++ {
			for j := 0; j < 256; j++ {
				l.bias[d*256+j] = float32(rawBias[d*512+j] + rawBias[d*512+256+j])
			}
		}
		b.layers[i] = l
	}
	return b, nil
}

// Predict executes the complete audited bou graph for one sequence and
// returns three class probabilities per input step.
func (b *BoundLayers) Predict(input []float32, steps int) ([]float32, error) {
	_, hidden, err := b.Run(input, steps)
	if err != nil {
		return nil, err
	}
	output := make([]float32, steps*3)
	for s := 0; s < steps; s++ {
		var logits [3]float32
		for j := 0; j < 3; j++ {
			var dot float32
			for k := 0; k < 128; k++ {
				dot = fma32(hidden[s*128+k], b.dense[k*3+j], dot)
			}
			logits[j] = float32(dot + b.bias[j])
			if logits[j] < 0 {
				logits[j] = 0
			}
		}
		prob := softmax3(logits)
		copy(output[s*3:], prob[:])
	}
	return output, nil
}

// Run accepts one sequence of steps*233 features and returns the two
// sequences in [steps, forward+reverse hidden] order (128 values per step).
// Initial hidden/cell states are zero, as in the audited graph.
func (b *BoundLayers) Run(input []float32, steps int) (first, second []float32, err error) {
	width := b.layers[0].width
	if steps <= 0 || len(input)%width != 0 || len(input)/width != steps {
		return nil, nil, fmt.Errorf("BOUND features require positive steps*%d values", width)
	}
	first = b.layers[0].run(input, steps)
	second = b.layers[1].run(first, steps)
	return first, second, nil
}

func (l *lstmLayer) run(input []float32, steps int) []float32 {
	h := l.hidden
	output := make([]float32, steps*2*h)
	for d := 0; d < 2; d++ {
		state, cell := make([]float32, h), make([]float32, h)
		gates := make([]float32, 4*h)
		for s := 0; s < steps; s++ {
			t := s
			if d == 1 {
				t = steps - 1 - s
			}
			for g := 0; g < 4*h; g++ {
				var x, recurrent float32
				w := l.w[(d*4*h+g)*l.width:]
				r := l.r[(d*4*h+g)*h:]
				for k := 0; k < l.width; k++ {
					x = fma32(input[t*l.width+k], w[k], x)
				}
				for k := 0; k < h; k++ {
					recurrent = fma32(state[k], r[k], recurrent)
				}
				gates[g] = float32(float32(x+recurrent) + l.bias[d*4*h+g])
			}
			// ONNX stores gates in input, output, forget, cell order.
			for j := 0; j < h; j++ {
				i, o, f, c := Sigmoid(gates[j]), Sigmoid(gates[h+j]), Sigmoid(gates[2*h+j]), Tanh(gates[3*h+j])
				cell[j] = float32(float32(cell[j]*f) + float32(i*c))
				state[j] = float32(Tanh(cell[j]) * o)
				output[t*2*h+d*h+j] = state[j]
			}
		}
	}
	return output
}
