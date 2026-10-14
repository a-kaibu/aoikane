package onnx

import (
	"fmt"
	"math"
	"slices"
)

func graphIndexNode(n graphNode, in []Value) ([]Value, bool, error) {
	fail := func(s string) ([]Value, bool, error) { return nil, true, fmt.Errorf("%s", s) }
	switch n.op {
	case "Neg":
		if len(in) != 1 || len(n.attrs) != 0 || (in[0].Kind != 1 && in[0].Kind != 7) {
			return fail("unsupported Neg input")
		}
		v := cloneValue(in[0])
		for i := range v.F {
			v.F[i] = -v.F[i]
		}
		for i := range v.I {
			v.I[i] = -v.I[i]
		}
		return []Value{v}, true, nil
	case "Flatten":
		if len(in) != 1 {
			return fail("Flatten requires one input")
		}
		for k := range n.attrs {
			if k != "axis" {
				return fail("unsupported Flatten attribute")
			}
		}
		axis := int64(1)
		if a, ok := n.attrs["axis"]; ok {
			axis = a.i
		}
		if axis < 0 {
			axis += int64(len(in[0].Shape))
		}
		if axis < 0 || axis > int64(len(in[0].Shape)) {
			return fail("invalid Flatten axis")
		}
		a, e := valueCount(in[0].Shape[:axis])
		if e != nil {
			return nil, true, e
		}
		b, e := valueCount(in[0].Shape[axis:])
		if e != nil {
			return nil, true, e
		}
		v := in[0]
		v.Shape = []int64{int64(a), int64(b)}
		return []Value{v}, true, nil
	case "ReduceMin", "ReduceMax":
		if len(in) != 1 || (in[0].Kind != 1 && in[0].Kind != 7 && in[0].Kind != 6) {
			return fail("unsupported extremum input")
		}
		for k, a := range n.attrs {
			if k != "keepdims" || a.i != 0 {
				return fail("only scalar all-axis extremum is implemented")
			}
		}
		if _, ok := n.attrs["keepdims"]; !ok {
			return fail("extremum requires explicit keepdims=0")
		}
		v := Value{Kind: in[0].Kind}
		if v.Kind == 1 {
			if len(in[0].F) == 0 {
				return fail("empty extremum input")
			}
			x := in[0].F[0]
			for _, y := range in[0].F[1:] {
				if math.IsNaN(float64(y)) || n.op == "ReduceMin" && y < x || n.op == "ReduceMax" && y > x {
					x = y
				}
			}
			v.F = []float32{x}
		} else {
			if len(in[0].I) == 0 {
				return fail("empty extremum input")
			}
			x := in[0].I[0]
			for _, y := range in[0].I[1:] {
				if n.op == "ReduceMin" && y < x || n.op == "ReduceMax" && y > x {
					x = y
				}
			}
			v.I = []int64{x}
		}
		return []Value{v}, true, nil
	case "ScatterND":
		if len(in) != 3 || in[1].Kind != 7 || in[0].Kind != in[2].Kind || len(in[1].Shape) < 1 {
			return fail("invalid ScatterND inputs")
		}
		for k, a := range n.attrs {
			if k != "reduction" || a.text != "none" {
				return fail("unsupported ScatterND reduction/attribute")
			}
		}
		for _, v := range in {
			if e := v.validate(); e != nil {
				return nil, true, e
			}
		}
		data, indices, updates := in[0], in[1], in[2]
		rank := len(data.Shape)
		k := int(indices.Shape[len(indices.Shape)-1])
		if k < 1 || k > rank {
			return fail("invalid ScatterND index depth")
		}
		shape := append(slices.Clone(indices.Shape[:len(indices.Shape)-1]), data.Shape[k:]...)
		if !slices.Equal(shape, updates.Shape) {
			return fail("ScatterND updates shape differs")
		}
		piece, e := valueCount(data.Shape[k:])
		if e != nil {
			return nil, true, e
		}
		v := cloneValue(data)
		for tuple := 0; tuple < len(indices.I)/k; tuple++ {
			offset := int64(0)
			for axis := 0; axis < k; axis++ {
				index := indices.I[tuple*k+axis]
				if index < 0 {
					index += data.Shape[axis]
				}
				if index < 0 || index >= data.Shape[axis] {
					return fail("ScatterND index out of range")
				}
				offset = offset*data.Shape[axis] + index
			}
			for j := 0; j < piece; j++ {
				copyElement(&v, int(offset)*piece+j, updates, tuple*piece+j)
			}
		}
		return []Value{v}, true, nil
	}
	return nil, false, nil
}
