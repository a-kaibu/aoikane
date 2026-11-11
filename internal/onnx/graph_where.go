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
	out, err := emptyValueUncleared(in[1].Kind, shape) // every element is selected
	if err != nil {
		return nil, err
	}
	count, _ := valueCount(shape)
	if same := len(in[0].B) == count && sameShape(in[0].Shape, shape) && sameShape(in[1].Shape, shape) && sameShape(in[2].Shape, shape); same && (in[1].Kind == 7 || in[1].Kind == 1) {
		cond := in[0].B
		parallelElements(count, func(lo, hi int) {
			if in[1].Kind == 7 {
				x, y, o := in[1].I, in[2].I, out.I
				for i := lo; i < hi; i++ {
					if cond[i] {
						o[i] = x[i]
					} else {
						o[i] = y[i]
					}
				}
				return
			}
			x, y, o := in[1].F, in[2].F, out.F
			for i := lo; i < hi; i++ {
				if cond[i] {
					o[i] = x[i]
				} else {
					o[i] = y[i]
				}
			}
		})
		return []Value{out}, nil
	}
	strides := [3][]int{shapeStrides(in[0].Shape), shapeStrides(in[1].Shape), shapeStrides(in[2].Shape)}
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

func sameShape(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
