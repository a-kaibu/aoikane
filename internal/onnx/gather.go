package onnx

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
)

// Gather is an ONNX graph containing one axis-zero Gather from an immutable
// rank-two FLOAT initializer. Rows are copied without floating-point arithmetic.
type Gather struct {
	Rows, Width int
	weights     []byte
}

func OpenGather(path string) (*Gather, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return loadGather(b)
}

func loadGather(data []byte) (*Gather, error) {
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
	nodeData, err := oneBytes(graph, 1)
	if err != nil {
		return nil, fmt.Errorf("single Gather node required: %w", err)
	}
	node, err := fields(nodeData)
	if err != nil {
		return nil, err
	}
	op, err := oneBytes(node, 4)
	if err != nil || string(op) != "Gather" {
		return nil, fmt.Errorf("only Gather is supported")
	}
	for _, f := range node {
		if f.number == 5 || (f.number == 7 && string(f.data) != "" && string(f.data) != "ai.onnx") {
			return nil, fmt.Errorf("Gather attributes/custom domains are unsupported")
		}
	}
	inputs, err := bytesFields(node, 1)
	if err != nil || len(inputs) != 2 {
		return nil, fmt.Errorf("Gather requires weight and indices inputs")
	}
	output, err := oneBytes(node, 2)
	if err != nil {
		return nil, err
	}
	for _, binding := range []struct {
		number uint64
		name   []byte
	}{{11, inputs[1]}, {12, output}} {
		valueData, err := oneBytes(graph, binding.number)
		if err != nil {
			return nil, err
		}
		value, err := fields(valueData)
		if err != nil {
			return nil, err
		}
		name, err := oneBytes(value, 1)
		if err != nil || string(name) != string(binding.name) {
			return nil, fmt.Errorf("Gather graph input/output binding mismatch")
		}
	}
	tensorData, err := oneBytes(graph, 5)
	if err != nil {
		return nil, err
	}
	tensor, err := fields(tensorData)
	if err != nil {
		return nil, err
	}
	name, err := oneBytes(tensor, 8)
	if err != nil || string(name) != string(inputs[0]) {
		return nil, fmt.Errorf("Gather weight binding mismatch")
	}
	var dims []uint64
	var dtype uint64
	for _, f := range tensor {
		switch f.number {
		case 1:
			if f.wire == 0 {
				dims = append(dims, f.value)
			} else if f.wire == 2 {
				for p := f.data; len(p) > 0; {
					d, n := binary.Uvarint(p)
					if n <= 0 {
						return nil, fmt.Errorf("invalid packed dimensions")
					}
					dims = append(dims, d)
					p = p[n:]
				}
			} else {
				return nil, fmt.Errorf("invalid dimension wire type")
			}
		case 2:
			if f.wire != 0 {
				return nil, fmt.Errorf("invalid tensor data type")
			}
			dtype = f.value
		case 13, 14:
			if f.number == 13 || f.value != 0 {
				return nil, fmt.Errorf("external weights are unsupported")
			}
		}
	}
	weights, err := oneBytes(tensor, 9)
	if err != nil {
		return nil, err
	}
	if dtype != 1 || len(dims) != 2 || dims[0] == 0 || dims[1] == 0 || dims[0] > uint64(len(weights))/4 || dims[1] > uint64(len(weights))/4/dims[0] || dims[0]*dims[1]*4 != uint64(len(weights)) {
		return nil, fmt.Errorf("Gather requires a rank-two raw FLOAT weight matrix")
	}
	return &Gather{Rows: int(dims[0]), Width: int(dims[1]), weights: weights}, nil
}

func (g *Gather) Lookup(ids []int64) ([]float32, error) {
	if len(ids) > int(^uint(0)>>1)/g.Width {
		return nil, fmt.Errorf("embedding output is too large")
	}
	out := make([]float32, len(ids)*g.Width)
	for i, id := range ids {
		if id < -int64(g.Rows) || id >= int64(g.Rows) {
			return nil, fmt.Errorf("embedding index %d is outside %d rows", id, g.Rows)
		}
		if id < 0 {
			id += int64(g.Rows)
		}
		row := g.weights[int(id)*g.Width*4:]
		for j := 0; j < g.Width; j++ {
			out[i*g.Width+j] = math.Float32frombits(binary.LittleEndian.Uint32(row[4*j:]))
		}
	}
	return out, nil
}
