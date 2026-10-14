package onnx

import "fmt"

// Shape operations needed by the exported FeatNet style graphs.
func executeShapeNode(n graphNode, in []Value) ([]Value, bool, error) {
	if out, handled, err := executeLayoutNode(n, in); handled {
		return out, handled, err
	}
	allowed := map[string][]string{"Shape": {"start", "end"}, "Size": {}, "Identity": {}, "Cast": {"to"}, "Squeeze": {}, "Slice": {}}
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
			return fail("unsupported shape attribute " + name)
		}
	}
	if len(in) == 0 {
		return fail("missing shape input")
	}
	v := in[0]
	var out Value
	switch n.op {
	case "Size":
		if len(in) != 1 {
			return fail("Size input count differs")
		}
		count, err := valueCount(v.Shape)
		if err != nil {
			return nil, true, err
		}
		out = Value{Kind: 7, I: []int64{int64(count)}}
	case "Shape":
		if len(in) != 1 {
			return fail("Shape input count differs")
		}
		rank := int64(len(v.Shape))
		start, end := int64(0), rank
		if a, ok := n.attrs["start"]; ok {
			start = a.i
		}
		if a, ok := n.attrs["end"]; ok {
			end = a.i
		}
		if start < 0 {
			start += rank
		}
		if end < 0 {
			end += rank
		}
		start = min(max(start, 0), rank)
		end = min(max(end, start), rank)
		out = Value{Kind: 7, Shape: []int64{end - start}, I: append([]int64(nil), v.Shape[start:end]...)}
	case "Identity":
		if len(in) != 1 {
			return fail("Identity input count differs")
		}
		out = v
	case "Cast":
		if len(in) != 1 {
			return fail("Cast input count differs")
		}
		a, ok := n.attrs["to"]
		if !ok {
			return fail("Cast requires target kind")
		}
		count, _ := valueCount(v.Shape)
		var err error
		out, err = emptyValue(int(a.i), v.Shape)
		if err != nil {
			return nil, true, err
		}
		for i := 0; i < count; i++ {
			var f float32
			var wide float64
			var integer int64
			var b bool
			switch v.Kind {
			case 1:
				f = v.F[i]
				wide = float64(f)
				integer = int64(f)
				b = f != 0
			case 11:
				wide = v.D[i]
				f = float32(wide)
				integer = int64(wide)
				b = wide != 0
			case 6, 7:
				integer = v.I[i]
				wide = float64(integer)
				f = float32(integer)
				b = integer != 0
			case 9:
				b = v.B[i]
				if b {
					f = 1
					wide = 1
					integer = 1
				}
			}
			switch out.Kind {
			case 1:
				out.F[i] = f
			case 11:
				out.D[i] = wide
			case 6:
				out.I[i] = int64(int32(integer))
			case 7:
				out.I[i] = integer
			case 9:
				out.B[i] = b
			}
		}
	case "Squeeze":
		if len(in) > 2 {
			return fail("Squeeze input count differs")
		}
		remove := make([]bool, len(v.Shape))
		if len(in) == 1 {
			for i, d := range v.Shape {
				remove[i] = d == 1
			}
		} else {
			axes := in[1]
			if axes.Kind != 7 || len(axes.Shape) != 1 {
				return fail("Squeeze axes must be INT64 vector")
			}
			for _, axis := range axes.I {
				if axis < 0 {
					axis += int64(len(v.Shape))
				}
				if axis < 0 || axis >= int64(len(v.Shape)) || remove[axis] || v.Shape[axis] != 1 {
					return fail("invalid Squeeze axis")
				}
				remove[axis] = true
			}
		}
		out = v
		out.Shape = nil
		for i, d := range v.Shape {
			if !remove[i] {
				out.Shape = append(out.Shape, d)
			}
		}
	case "Slice":
		if len(in) < 3 || len(in) > 5 {
			return fail("Slice input count differs")
		}
		for _, x := range in[1:] {
			if x.Kind != 7 || len(x.Shape) != 1 {
				return fail("Slice indices must be INT64 vectors")
			}
		}
		starts, ends := in[1].I, in[2].I
		if len(starts) != len(ends) {
			return fail("Slice indices differ in length")
		}
		axes, steps := make([]int64, len(starts)), make([]int64, len(starts))
		for i := range axes {
			axes[i] = int64(i)
			steps[i] = 1
		}
		if len(in) >= 4 {
			axes = in[3].I
		}
		if len(in) == 5 {
			steps = in[4].I
		}
		if len(axes) != len(starts) || len(steps) != len(starts) {
			return fail("Slice axes or steps differ in length")
		}
		shape := append([]int64(nil), v.Shape...)
		origin := make([]int64, len(shape))
		stride := make([]int64, len(shape))
		seen := make([]bool, len(shape))
		for i := range stride {
			stride[i] = 1
		}
		for i, axis := range axes {
			if axis < 0 {
				axis += int64(len(shape))
			}
			if axis < 0 || axis >= int64(len(shape)) || seen[axis] || steps[i] == 0 {
				return fail("invalid Slice axis or step")
			}
			seen[axis] = true
			d, start, end, step := v.Shape[axis], starts[i], ends[i], steps[i]
			if start < 0 {
				start += d
			}
			if end < 0 {
				end += d
			}
			length := int64(0)
			if step > 0 {
				start = min(max(start, 0), d)
				end = min(max(end, 0), d)
				if end > start {
					length = 1 + (end-start-1)/step
				}
			} else {
				// Reject MinInt64 instead of overflowing when negating the step.
				if step == -1<<63 {
					return fail("Slice step overflows")
				}
				start = min(max(start, 0), d-1)
				end = min(max(end, -1), d-1)
				if start > end {
					length = 1 + (start-end-1)/(-step)
				}
			}
			shape[axis], origin[axis], stride[axis] = length, start, step
		}
		var err error
		out, err = emptyValue(v.Kind, shape)
		if err != nil {
			return nil, true, err
		}
		count, _ := valueCount(shape)
		inputStrides := shapeStrides(v.Shape)
		for i := 0; i < count; i++ {
			offset, remaining := int64(0), i
			for d := len(shape) - 1; d >= 0; d-- {
				coord := int64(remaining % int(shape[d]))
				remaining /= int(shape[d])
				offset += (origin[d] + coord*stride[d]) * int64(inputStrides[d])
			}
			copyElement(&out, i, v, int(offset))
		}
	}
	return []Value{out}, true, nil
}
