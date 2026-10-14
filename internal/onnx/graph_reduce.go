package onnx

import (
	"fmt"
	"math"
)

func graphMeanPair(n graphNode, in []Value) ([]Value, error) {
	if len(in) == 1 && in[0].Kind == 1 {
		for name := range n.attrs {
			if name != "keepdims" && name != "noop_with_empty_axes" {
				return nil, fmt.Errorf("unsupported global ReduceMean attribute")
			}
		}
		keep := int64(1)
		if a, ok := n.attrs["keepdims"]; ok {
			keep = a.i
		}
		if (keep != 0 && keep != 1) || n.attrs["noop_with_empty_axes"].i != 0 {
			return nil, fmt.Errorf("unsupported global ReduceMean flags")
		}
		mean, err := torchMeanAll(in[0].F)
		if err != nil {
			return nil, err
		}
		var shape []int64
		if keep != 0 {
			shape = make([]int64, len(in[0].Shape))
			for i := range shape {
				shape[i] = 1
			}
		}
		return []Value{{Kind: 1, Shape: shape, F: []float32{mean}}}, nil
	}
	if len(in) != 2 || in[0].Kind != 1 || in[1].Kind != 7 || len(in[1].Shape) != 1 || len(in[1].I) != 1 {
		return nil, fmt.Errorf("ReduceMean requires one axis and FLOAT input")
	}
	for name := range n.attrs {
		if name != "keepdims" && name != "noop_with_empty_axes" {
			return nil, fmt.Errorf("unsupported ReduceMean attribute")
		}
	}
	keep := int64(1)
	if a, ok := n.attrs["keepdims"]; ok {
		keep = a.i
	}
	noop := n.attrs["noop_with_empty_axes"].i
	if (keep != 0 && keep != 1) || (noop != 0 && noop != 1) {
		return nil, fmt.Errorf("invalid ReduceMean flags")
	}
	v := in[0]
	axis := in[1].I[0]
	if axis < 0 {
		axis += int64(len(v.Shape))
	}
	if axis < 0 || axis >= int64(len(v.Shape)) || v.Shape[axis] != 2 {
		return nil, fmt.Errorf("only two-element ReduceMean axis implemented")
	}
	shape := append([]int64(nil), v.Shape...)
	if keep != 0 {
		shape[axis] = 1
	} else {
		shape = append(shape[:axis], shape[axis+1:]...)
	}
	out, err := emptyValue(1, shape)
	if err != nil {
		return nil, err
	}
	outer, _ := valueCount(v.Shape[:axis])
	inner, _ := valueCount(v.Shape[axis+1:])
	for o := 0; o < outer; o++ {
		for k := 0; k < inner; k++ {
			sum := float32(v.F[o*2*inner+k] + v.F[(o*2+1)*inner+k])
			out.F[o*inner+k] = float32(sum / 2)
		}
	}
	return []Value{out}, nil
}

func graphIntegerReduce(n graphNode, in []Value) ([]Value, error) {
	if len(in) < 1 || len(in) > 2 || (in[0].Kind != 6 && in[0].Kind != 7) {
		return nil, fmt.Errorf("ReduceSum requires integer input; FLOAT reduction is not yet implemented")
	}
	for name := range n.attrs {
		if name != "keepdims" && name != "noop_with_empty_axes" {
			return nil, fmt.Errorf("unsupported ReduceSum attribute %s", name)
		}
	}
	keep := int64(1)
	if a, ok := n.attrs["keepdims"]; ok {
		keep = a.i
	}
	noop := n.attrs["noop_with_empty_axes"].i
	if (keep != 0 && keep != 1) || (noop != 0 && noop != 1) {
		return nil, fmt.Errorf("invalid ReduceSum flags")
	}
	v := in[0]
	var axes []int64
	if len(in) == 2 && in[1].Kind != 0 {
		if in[1].Kind != 7 || len(in[1].Shape) != 1 {
			return nil, fmt.Errorf("ReduceSum axes must be INT64 vector")
		}
		axes = in[1].I
	}
	if len(axes) == 0 && noop != 0 {
		return []Value{v}, nil
	}
	remove := make([]bool, len(v.Shape))
	if len(axes) == 0 {
		for i := range remove {
			remove[i] = true
		}
	} else {
		for _, a := range axes {
			if a < 0 {
				a += int64(len(remove))
			}
			if a < 0 || a >= int64(len(remove)) || remove[a] {
				return nil, fmt.Errorf("invalid ReduceSum axis")
			}
			remove[a] = true
		}
	}
	var shape []int64
	mapping := make([]int, len(v.Shape))
	for i, d := range v.Shape {
		mapping[i] = -1
		if !remove[i] {
			mapping[i] = len(shape)
			shape = append(shape, d)
		} else if keep != 0 {
			shape = append(shape, 1)
		}
	}
	out, err := emptyValue(v.Kind, shape)
	if err != nil {
		return nil, err
	}
	strides := shapeStrides(shape)
	for i, x := range v.I {
		index, remaining := 0, i
		for d := len(v.Shape) - 1; d >= 0; d-- {
			coord := remaining % int(v.Shape[d])
			remaining /= int(v.Shape[d])
			if mapping[d] >= 0 {
				index += coord * strides[mapping[d]]
			}
		}
		out.I[index] += x
		if v.Kind == 6 {
			out.I[index] = int64(int32(out.I[index]))
		}
	}
	return []Value{out}, nil
}

func graphIntegerRange(n graphNode, in []Value) ([]Value, error) {
	if len(n.attrs) != 0 || len(in) != 3 {
		return nil, fmt.Errorf("invalid Range inputs or attributes")
	}
	if in[0].Kind == 1 {
		integers := make([]Value, 3)
		for i, v := range in {
			if v.Kind != 1 || len(v.Shape) != 0 || len(v.F) != 1 || math.IsNaN(float64(v.F[0])) || math.Abs(float64(v.F[0])) > 1<<24 || math.Trunc(float64(v.F[0])) != float64(v.F[0]) {
				return nil, fmt.Errorf("FLOAT Range requires exactly representable integer scalars")
			}
			integers[i] = Value{Kind: 7, I: []int64{int64(v.F[0])}}
		}
		out, e := graphIntegerRange(n, integers)
		if e != nil {
			return nil, e
		}
		v := Value{Kind: 1, Shape: out[0].Shape, F: make([]float32, len(out[0].I))}
		for i, x := range out[0].I {
			v.F[i] = float32(x)
		}
		return []Value{v}, nil
	}
	kind := in[0].Kind
	for _, v := range in {
		if v.Kind != kind || (kind != 6 && kind != 7) || len(v.I) != 1 || len(v.Shape) != 0 {
			return nil, fmt.Errorf("Range requires integer scalars")
		}
	}
	start, end, step := in[0].I[0], in[1].I[0], in[2].I[0]
	if step == 0 {
		return nil, fmt.Errorf("Range step must be nonzero")
	}
	var gap, divisor uint64
	if step > 0 && start < end {
		gap, divisor = uint64(end)-uint64(start), uint64(step)
	} else if step < 0 && start > end {
		gap, divisor = uint64(start)-uint64(end), uint64(-step)
	}
	count := uint64(0)
	if divisor != 0 {
		count = gap / divisor
		if gap%divisor != 0 {
			count++
		}
	}
	if count > uint64(^uint(0)>>1) {
		return nil, fmt.Errorf("Range length overflows")
	}
	out, err := emptyValue(kind, []int64{int64(count)})
	if err != nil {
		return nil, err
	}
	x := start
	for i := range out.I {
		out.I[i] = x
		if i+1 < len(out.I) {
			x += step
		}
	}
	return []Value{out}, nil
}
