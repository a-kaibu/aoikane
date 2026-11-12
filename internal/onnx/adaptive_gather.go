package onnx

import (
	"math"
	"slices"
)

type adaptiveGatherPlan struct{ inputs [4]string }

// prepareAdaptiveGathers recognizes the audited export's channel-wise gather:
// Reshape(Reshape(Gather(Flatten(Pad(x)), flattened_indices))). The original
// flattened indices include a batch grid and channel grid, constructed through
// thousands of ScatterND/Expand nodes. Batch is one and channels are independent,
// so only the rounded temporal indices and the zero-padding extents are needed.
// The caller must verify the model digest before enabling this plan.
func (g *Graph) prepareAdaptiveGathers() {
	parent := func(name, op string) *graphNode {
		i, ok := g.producer[name]
		if !ok || g.nodes[i].op != op {
			return nil
		}
		return &g.nodes[i]
	}
	plans := make(map[int]adaptiveGatherPlan)
	for i, n := range g.nodes {
		if n.op != "Reshape" || len(n.inputs) != 2 || len(n.outputs) != 1 {
			continue
		}
		inner := parent(n.inputs[0], "Reshape")
		if inner == nil || len(inner.inputs) != 2 {
			continue
		}
		gather := parent(inner.inputs[0], "Gather")
		if gather == nil || len(gather.inputs) != 2 {
			continue
		}
		flat := parent(gather.inputs[0], "Flatten")
		if flat == nil || len(flat.inputs) != 1 {
			continue
		}
		pad := parent(flat.inputs[0], "Pad")
		if pad == nil || len(pad.inputs) != 3 {
			continue
		}
		indices := make(map[string]bool)
		visited := make(map[string]bool)
		var visit func(string)
		visit = func(name string) {
			if visited[name] {
				return
			}
			visited[name] = true
			idx, ok := g.producer[name]
			if !ok {
				return
			}
			p := g.nodes[idx]
			switch p.op {
			case "Shape", "ScatterND", "Constant", "ReduceMin", "ReduceMax":
				return
			case "Cast":
				if len(p.inputs) == 1 && parent(p.inputs[0], "Round") != nil {
					indices[name] = true
					return
				}
			}
			for _, key := range p.inputs {
				visit(key)
			}
		}
		visit(gather.inputs[1])
		if len(indices) != 1 {
			continue
		}
		for index := range indices {
			plans[i] = adaptiveGatherPlan{inputs: [4]string{pad.inputs[0], index, pad.inputs[1], pad.inputs[2]}}
		}
	}
	// A different or partially recognized export must retain ordinary execution.
	if len(plans) == 30 {
		g.adaptive = plans
	}
}

func runAdaptiveGather(in []Value) (Value, bool) {
	if len(in) != 4 {
		return Value{}, false
	}
	x, idx, pads, fill := in[0], in[1], in[2], in[3]
	if x.Kind != 1 || idx.Kind != 7 || pads.Kind != 7 || fill.Kind != 1 || len(x.Shape) != 3 || x.Shape[0] != 1 || x.Shape[1] < 1 || x.Shape[2] < 1 || !slices.Equal(idx.Shape, []int64{1, 1, x.Shape[2]}) || !slices.Equal(pads.Shape, []int64{6}) || len(fill.F) != 1 || math.Float32bits(fill.F[0]) != 0 {
		return Value{}, false
	}
	if x.validate() != nil || idx.validate() != nil || pads.validate() != nil {
		return Value{}, false
	}
	if pads.I[0] != 0 || pads.I[1] != 0 || pads.I[3] != 0 || pads.I[4] != 0 || pads.I[2] < 0 || pads.I[5] < 0 || pads.I[2] > math.MaxInt32 || pads.I[5] > math.MaxInt32 {
		return Value{}, false
	}
	frames, channels := int(x.Shape[2]), int(x.Shape[1])
	if frames > math.MaxInt32 || len(idx.I) != frames {
		return Value{}, false
	}
	left, padded := pads.I[2], pads.I[2]+int64(frames)+pads.I[5]
	positions := make([]int32, frames)
	for i, index := range idx.I {
		if index < 0 {
			index += padded
		}
		if index < 0 || index >= padded {
			return Value{}, false
		}
		pos := index - left
		positions[i] = -1
		if pos >= 0 && pos < int64(frames) {
			positions[i] = int32(pos)
		}
	}
	y := getFloatsUncleared(len(x.F))
	if !fastGatherBulk(y, x.F, positions, channels, frames) {
		for c := 0; c < channels; c++ {
			src, dst := x.F[c*frames:(c+1)*frames], y[c*frames:(c+1)*frames]
			for t, pos := range positions {
				dst[t] = 0
				if pos >= 0 {
					dst[t] = src[pos]
				}
			}
		}
	}
	return Value{Kind: 1, Shape: slices.Clone(x.Shape), F: y}, true
}
