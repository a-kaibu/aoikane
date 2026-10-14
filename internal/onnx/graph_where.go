package onnx

import "fmt"

func broadcastShape(a, b []int64) ([]int64, error) {
	rank := max(len(a), len(b))
	shape := make([]int64, rank)
	for i := range shape {
		x, y := int64(1), int64(1)
		if d := i - rank + len(a); d >= 0 {
			x = a[d]
		}
		if d := i - rank + len(b); d >= 0 {
			y = b[d]
		}
		if x == y || y == 1 {
			shape[i] = x
		} else if x == 1 {
			shape[i] = y
		} else {
			return nil, fmt.Errorf("input shapes do not broadcast")
		}
	}
	return shape, nil
}

func graphWhere(n graphNode, in []Value) ([]Value, error) {
	if len(n.attrs) != 0 || len(in) != 3 || in[0].Kind != 9 || in[1].Kind != in[2].Kind {
		return nil, fmt.Errorf("invalid Where inputs or attributes")
	}
	shape, err := broadcastShape(in[1].Shape, in[2].Shape)
	if err != nil {
		return nil, err
	}
	shape, err = broadcastShape(shape, in[0].Shape)
	if err != nil {
		return nil, err
	}
	out, err := emptyValue(in[1].Kind, shape)
	if err != nil {
		return nil, err
	}
	strides := [3][]int{shapeStrides(in[0].Shape), shapeStrides(in[1].Shape), shapeStrides(in[2].Shape)}
	count, _ := valueCount(shape)
	for i := 0; i < count; i++ {
		indices := [3]int{}
		remaining := i
		for d := len(shape) - 1; d >= 0; d-- {
			coord := remaining % int(shape[d])
			remaining /= int(shape[d])
			for k, v := range in {
				if s := d - len(shape) + len(v.Shape); s >= 0 && v.Shape[s] != 1 {
					indices[k] += coord * strides[k][s]
				}
			}
		}
		which := 2
		if in[0].B[indices[0]] {
			which = 1
		}
		copyElement(&out, i, in[which], indices[which])
	}
	return []Value{out}, nil
}
