package onnx

import (
	"fmt"
	"os"
)

// Linear executes a MatMul with an immutable rank-two FLOAT right operand.
// It accepts a contiguous sequence of rows, without batch broadcasting.
type Linear struct {
	In, Out int
	weight  []float32
}

func OpenLinear(path, name string) (*Linear, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	model, err := fields(data)
	if err != nil {
		return nil, err
	}
	gd, err := oneBytes(model, 7)
	if err != nil {
		return nil, err
	}
	graph, err := fields(gd)
	if err != nil {
		return nil, err
	}
	nodes, err := bytesFields(graph, 1)
	if err != nil {
		return nil, err
	}
	var node []field
	for _, nd := range nodes {
		n, err := fields(nd)
		if err != nil {
			return nil, err
		}
		names, err := bytesFields(n, 3)
		if err != nil {
			return nil, err
		}
		if len(names) == 1 && string(names[0]) == name {
			if node != nil {
				return nil, fmt.Errorf("duplicate MatMul node")
			}
			node = n
		}
	}
	if node == nil {
		return nil, fmt.Errorf("MatMul node %s not found", name)
	}
	op, err := oneBytes(node, 4)
	if err != nil || string(op) != "MatMul" {
		return nil, fmt.Errorf("selected node is not MatMul")
	}
	for _, f := range node {
		if f.number == 5 || (f.number == 7 && len(f.data) > 0 && string(f.data) != "ai.onnx") {
			return nil, fmt.Errorf("MatMul attributes/custom domains unsupported")
		}
	}
	inputs, err := bytesFields(node, 1)
	if err != nil || len(inputs) != 2 {
		return nil, fmt.Errorf("MatMul requires two inputs")
	}
	initializers, err := bytesFields(graph, 5)
	if err != nil {
		return nil, err
	}
	for _, td := range initializers {
		t, err := fields(td)
		if err != nil {
			return nil, err
		}
		tn, err := oneBytes(t, 8)
		if err != nil {
			return nil, err
		}
		if string(tn) != string(inputs[1]) {
			continue
		}
		w, dims, err := rawFloatTensor(t)
		if err != nil {
			return nil, err
		}
		if len(dims) != 2 {
			return nil, fmt.Errorf("MatMul requires a rank-two right operand")
		}
		if dims[0] > 256 {
			return nil, fmt.Errorf("MatMul inner dimension above 256 is not yet supported")
		}
		return &Linear{In: dims[0], Out: dims[1], weight: w}, nil
	}
	return nil, fmt.Errorf("constant MatMul right operand not found")
}

func (l *Linear) Run(input []float32, rows int) ([]float32, error) {
	if rows <= 0 || len(input)%l.In != 0 || len(input)/l.In != rows {
		return nil, fmt.Errorf("MatMul input shape differs")
	}
	if rows > int(^uint(0)>>1)/l.Out {
		return nil, fmt.Errorf("MatMul output too large")
	}
	out := make([]float32, rows*l.Out)
	for r := 0; r < rows; r++ {
		for j := 0; j < l.Out; j++ {
			var dot float32
			for k := 0; k < l.In; k++ {
				dot = fma32(input[r*l.In+k], l.weight[k*l.Out+j], dot)
			}
			out[r*l.Out+j] = dot
		}
	}
	return out, nil
}
