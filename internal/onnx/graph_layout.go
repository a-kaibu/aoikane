package onnx

import "fmt"

func executeLayoutNode(n graphNode, in []Value) ([]Value, bool, error) {
	allowed := map[string][]string{"Reshape": {"allowzero"}, "Expand": {}, "Tile": {}, "Concat": {"axis"}, "Split": {"axis"}}
	attrs, handled := allowed[n.op]
	if !handled {
		return nil, false, nil
	}
	fail := func(s string) ([]Value, bool, error) { return nil, true, fmt.Errorf("%s", s) }
	for name := range n.attrs {
		found := false
		for _, a := range attrs {
			found = found || a == name
		}
		if !found {
			return fail("unsupported layout attribute " + name)
		}
	}
	if len(in) == 0 {
		return fail("missing layout input")
	}
	v := in[0]
	if n.op == "Concat" || n.op == "Split" {
		axis := n.attrs["axis"].i
		if n.op == "Concat" {
			if _, ok := n.attrs["axis"]; !ok {
				return fail("Concat requires axis")
			}
		}
		if axis < 0 {
			axis += int64(len(v.Shape))
		}
		if axis < 0 || axis >= int64(len(v.Shape)) {
			return fail("invalid layout axis")
		}
		a := int(axis)
		shape := append([]int64(nil), v.Shape...)
		outer, _ := valueCount(shape[:a])
		inner, _ := valueCount(shape[a+1:])
		if n.op == "Concat" {
			shape[a] = 0
			for _, x := range in {
				if x.Kind != v.Kind || len(x.Shape) != len(shape) {
					return fail("Concat kinds or ranks differ")
				}
				for d := range shape {
					if d != a && x.Shape[d] != shape[d] {
						return fail("Concat shapes differ")
					}
				}
				if x.Shape[a] > int64(^uint(0)>>1)-shape[a] {
					return fail("Concat dimension overflows")
				}
				shape[a] += x.Shape[a]
			}
			out, err := emptyValue(v.Kind, shape)
			if err != nil {
				return nil, true, err
			}
			at := 0
			for o := 0; o < outer; o++ {
				for _, x := range in {
					width := int(x.Shape[a]) * inner
					for k := 0; k < width; k++ {
						copyElement(&out, at, x, o*width+k)
						at++
					}
				}
			}
			return []Value{out}, true, nil
		}
		if len(in) > 2 || len(n.outputs) == 0 {
			return fail("invalid Split inputs or outputs")
		}
		sizes := make([]int64, len(n.outputs))
		if len(in) == 2 {
			if in[1].Kind != 7 || len(in[1].Shape) != 1 || len(in[1].I) != len(sizes) {
				return fail("Split sizes must match outputs")
			}
			copy(sizes, in[1].I)
		} else {
			if shape[a]%int64(len(sizes)) != 0 {
				return fail("Split axis is not evenly divisible")
			}
			for i := range sizes {
				sizes[i] = shape[a] / int64(len(sizes))
			}
		}
		total := int64(0)
		for _, s := range sizes {
			if s < 0 || s > shape[a]-total {
				return fail("invalid Split sizes")
			}
			total += s
		}
		if total != shape[a] {
			return fail("Split sizes do not cover axis")
		}
		out := make([]Value, len(sizes))
		offset := 0
		for i, size := range sizes {
			shape[a] = size
			var err error
			out[i], err = emptyValue(v.Kind, shape)
			if err != nil {
				return nil, true, err
			}
			width := int(size) * inner
			for o := 0; o < outer; o++ {
				for k := 0; k < width; k++ {
					copyElement(&out[i], o*width+k, v, o*int(v.Shape[a])*inner+offset+k)
				}
			}
			offset += width
		}
		return out, true, nil
	}
	if len(in) != 2 || in[1].Kind != 7 || len(in[1].Shape) != 1 {
		return fail("layout requires INT64 shape vector")
	}
	shape := append([]int64(nil), in[1].I...)
	if n.op == "Reshape" {
		allowzero := n.attrs["allowzero"].i
		if allowzero != 0 && allowzero != 1 {
			return fail("invalid Reshape allowzero")
		}
		inferred := -1
		for d, x := range shape {
			if x == -1 {
				if inferred >= 0 {
					return fail("multiple inferred Reshape dimensions")
				}
				inferred = d
				shape[d] = 1
			} else if x < 0 {
				return fail("invalid Reshape dimension")
			} else if x == 0 && allowzero == 0 {
				if d >= len(v.Shape) {
					return fail("Reshape zero outside input rank")
				}
				shape[d] = v.Shape[d]
			}
		}
		count, err := valueCount(shape)
		if err != nil {
			return nil, true, err
		}
		inputCount, _ := valueCount(v.Shape)
		if inferred >= 0 {
			if count == 0 || inputCount%count != 0 {
				return fail("Reshape cannot infer dimension")
			}
			shape[inferred] = int64(inputCount / count)
		} else if count != inputCount {
			return fail("Reshape element count differs")
		}
		v.Shape = shape
		return []Value{v}, true, nil
	}
	if n.op == "Tile" {
		if len(shape) != len(v.Shape) {
			return fail("Tile repeat rank differs")
		}
		for i, r := range shape {
			if r < 0 || (v.Shape[i] != 0 && r > int64(^uint(0)>>1)/v.Shape[i]) {
				return fail("invalid or overflowing Tile repeats")
			}
			shape[i] = r * v.Shape[i]
		}
	} else {
		for _, d := range shape {
			if d < 0 {
				return fail("negative Expand dimension")
			}
		}
		rank := max(len(shape), len(v.Shape))
		target := make([]int64, rank)
		for i := range target {
			x, y := int64(1), int64(1)
			if d := i - rank + len(v.Shape); d >= 0 {
				x = v.Shape[d]
			}
			if d := i - rank + len(shape); d >= 0 {
				y = shape[d]
			}
			if x == y || y == 1 {
				target[i] = x
			} else if x == 1 {
				target[i] = y
			} else {
				return fail("Expand shapes do not broadcast")
			}
		}
		shape = target
	}
	out, err := emptyValue(v.Kind, shape)
	if err != nil {
		return nil, true, err
	}
	count, _ := valueCount(shape)
	strides := shapeStrides(v.Shape)
	padding := len(shape) - len(v.Shape)
	for i := 0; i < count; i++ {
		offset, remaining := 0, i
		for d := len(shape) - 1; d >= 0; d-- {
			coord := remaining % int(shape[d])
			remaining /= int(shape[d])
			if s := d - padding; s >= 0 {
				offset += (coord % int(v.Shape[s])) * strides[s]
			}
		}
		copyElement(&out, i, v, offset)
	}
	return []Value{out}, true, nil
}
