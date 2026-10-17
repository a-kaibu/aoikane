package onnx

import (
	"fmt"
	"math"
)

func executeSequenceNode(n graphNode, in []Value, original bool) ([]Value, bool, error) {
	allowed := map[string][]string{"CumSum": {"exclusive", "reverse"}, "Round": {}, "ConstantOfShape": {"value"}, "Pad": {"mode"}}
	attrs, ok := allowed[n.op]
	if !ok {
		return nil, false, nil
	}
	fail := func(s string) ([]Value, bool, error) { return nil, true, fmt.Errorf("%s", s) }
	for name := range n.attrs {
		found := false
		for _, a := range attrs {
			found = found || a == name
		}
		if !found {
			return fail("unsupported sequence attribute " + name)
		}
	}
	if len(in) == 0 {
		return fail("missing sequence input")
	}
	v := in[0]
	var out Value
	var err error
	switch n.op {
	case "Round":
		if len(in) != 1 || v.Kind != 1 {
			return fail("Round requires FLOAT input")
		}
		out = cloneValue(v)
		for i, x := range out.F {
			out.F[i] = float32(math.RoundToEven(float64(x)))
		}
	case "CumSum":
		if len(in) != 2 || (in[1].Kind != 6 && in[1].Kind != 7) || len(in[1].I) != 1 || len(v.Shape) == 0 || (v.Kind != 1 && v.Kind != 6 && v.Kind != 7 && v.Kind != 11) {
			return fail("invalid CumSum inputs")
		}
		exclusive, reverse := n.attrs["exclusive"].i, n.attrs["reverse"].i
		if (exclusive != 0 && exclusive != 1) || (reverse != 0 && reverse != 1) {
			return fail("invalid CumSum flags")
		}
		axis := in[1].I[0]
		if axis < 0 {
			axis += int64(len(v.Shape))
		}
		if axis < 0 || axis >= int64(len(v.Shape)) {
			return fail("invalid CumSum axis")
		}
		outer, _ := valueCount(v.Shape[:axis])
		inner, _ := valueCount(v.Shape[axis+1:])
		width := int(v.Shape[axis])
		out, err = emptyValue(v.Kind, v.Shape)
		if err != nil {
			return nil, true, err
		}
		for o := 0; o < outer; o++ {
			for k := 0; k < inner; k++ {
				var sum float32
				var wide float64
				var integer int64
				for j := 0; j < width; j++ {
					d := j
					if reverse != 0 {
						d = width - 1 - j
					}
					index := (o*width+d)*inner + k
					if v.Kind == 1 {
						x := v.F[index]
						// ATen's CPU FLOAT cumsum accumulates in double, then
						// rounds each output to float. Rounding the accumulator
						// itself can move a phoneme across a frame boundary.
						if original {
							if exclusive != 0 {
								out.F[index] = float32(wide)
							}
							wide += float64(x)
							if exclusive == 0 {
								out.F[index] = float32(wide)
							}
							continue
						}
						if exclusive != 0 {
							out.F[index] = sum
						}
						sum = float32(sum + x)
						if exclusive == 0 {
							out.F[index] = sum
						}
					} else if v.Kind == 11 {
						if exclusive != 0 {
							out.D[index] = wide
						}
						wide = float64(wide + v.D[index])
						if exclusive == 0 {
							out.D[index] = wide
						}
					} else {
						x := v.I[index]
						if exclusive != 0 {
							out.I[index] = integer
						}
						integer += x
						if v.Kind == 6 {
							integer = int64(int32(integer))
						}
						if exclusive == 0 {
							out.I[index] = integer
						}
					}
				}
			}
		}
	case "ConstantOfShape":
		if len(in) != 1 || v.Kind != 7 || len(v.Shape) != 1 {
			return fail("ConstantOfShape requires INT64 vector")
		}
		fill := Value{Kind: 1, F: []float32{0}}
		if a, ok := n.attrs["value"]; ok {
			if a.tensor == nil {
				return fail("ConstantOfShape requires tensor value")
			}
			fill = *a.tensor
		}
		count, err := valueCount(fill.Shape)
		if err != nil || count != 1 {
			return fail("ConstantOfShape value must have one element")
		}
		out, err = emptyValue(fill.Kind, v.I)
		if err != nil {
			return nil, true, err
		}
		count, _ = valueCount(v.I)
		for i := 0; i < count; i++ {
			copyElement(&out, i, fill, 0)
		}
	case "Pad":
		if len(in) < 2 || len(in) > 3 || in[1].Kind != 7 || len(in[1].Shape) != 1 || len(in[1].I) != 2*len(v.Shape) {
			return fail("invalid Pad inputs")
		}
		if mode, ok := n.attrs["mode"]; ok && mode.text != "constant" {
			return fail("unsupported Pad mode")
		}
		fill, err := emptyValue(v.Kind, nil)
		if err != nil {
			return nil, true, err
		}
		if len(in) == 3 && in[2].Kind != 0 {
			fill = in[2]
			count, err := valueCount(fill.Shape)
			if err != nil || count != 1 || fill.Kind != v.Kind {
				return fail("Pad constant must match input kind")
			}
		}
		shape := append([]int64(nil), v.Shape...)
		rank := len(shape)
		for i := range shape {
			left, right := in[1].I[i], in[1].I[i+rank]
			if left < -shape[i] || right < -shape[i] {
				return fail("Pad crop exceeds input dimension")
			}
			cropLeft, cropRight := max(-left, int64(0)), max(-right, int64(0))
			if cropRight > shape[i]-cropLeft {
				return fail("Pad crops overlap")
			}
			shape[i] -= cropLeft + cropRight
			left, right = max(left, int64(0)), max(right, int64(0))
			limit := int64(^uint(0) >> 1)
			if left > limit-shape[i] || right > limit-shape[i]-left {
				return fail("Pad dimension overflows")
			}
			shape[i] += left + right
		}
		out, err = emptyValue(v.Kind, shape)
		if err != nil {
			return nil, true, err
		}
		count, _ := valueCount(shape)
		strides := shapeStrides(v.Shape)
		for i := 0; i < count; i++ {
			remaining, offset, inside := i, 0, true
			for d := rank - 1; d >= 0; d-- {
				coord := int64(remaining%int(shape[d])) - in[1].I[d]
				remaining /= int(shape[d])
				if coord < 0 || coord >= v.Shape[d] {
					inside = false
				} else {
					offset += int(coord) * strides[d]
				}
			}
			if inside {
				copyElement(&out, i, v, offset)
			} else {
				copyElement(&out, i, fill, 0)
			}
		}
	}
	return []Value{out}, true, nil
}
