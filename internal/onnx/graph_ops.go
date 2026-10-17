package onnx

import (
	"fmt"
	"math"
)

func emptyValue(kind int, shape []int64) (Value, error) {
	n, err := valueCount(shape)
	if err != nil {
		return Value{}, err
	}
	v := Value{Kind: kind, Shape: append([]int64(nil), shape...)}
	switch kind {
	case 1:
		v.F = make([]float32, n)
	case 11:
		v.D = make([]float64, n)
	case 6, 7:
		v.I = make([]int64, n)
	case 9:
		v.B = make([]bool, n)
	default:
		return v, fmt.Errorf("unsupported value kind")
	}
	return v, nil
}
func copyElement(dst *Value, di int, src Value, si int) {
	switch dst.Kind {
	case 1:
		dst.F[di] = src.F[si]
	case 11:
		dst.D[di] = src.D[si]
	case 6, 7:
		dst.I[di] = src.I[si]
	case 9:
		dst.B[di] = src.B[si]
	}
}

func executeNode(n graphNode, in []Value, original bool, threads ...int) ([]Value, error) {
	if out, handled, err := graphIndexNode(n, in); handled {
		return out, err
	}
	if n.op == "Sin" || n.op == "Tanh" {
		if len(n.attrs) != 0 || len(in) != 1 || in[0].Kind != 1 {
			return nil, fmt.Errorf("%s requires FLOAT input and no attributes", n.op)
		}
		v := cloneValue(in[0])
		for i, x := range v.F {
			if n.op == "Sin" {
				if original && originalAVX512Unary(threads) {
					v.F[i] = MKLSinAVX512(x)
				} else {
					v.F[i] = MKLSin(x)
				}
			} else if original {
				if originalAVX512Unary(threads) {
					v.F[i] = MKLTanhAVX512(x)
				} else {
					v.F[i] = MKLTanh(x)
				}
			} else {
				v.F[i] = Tanh(x)
			}
		}
		return []Value{v}, nil
	}
	if n.op == "Resize" {
		return graphResizeNearest1D(n, in)
	}
	if n.op == "Floor" {
		if len(n.attrs) != 0 || len(in) != 1 || in[0].Kind != 1 {
			return nil, fmt.Errorf("Floor requires FLOAT input and no attributes")
		}
		v := cloneValue(in[0])
		for i, x := range v.F {
			v.F[i] = float32(math.Floor(float64(x)))
		}
		return []Value{v}, nil
	}
	if n.op == "AveragePool" {
		return graphAveragePool(n, in)
	}
	if n.op == "Log" {
		if len(n.attrs) != 0 || len(in) != 1 || in[0].Kind != 1 {
			return nil, fmt.Errorf("Log requires FLOAT input and no attributes")
		}
		var values []float32
		if original {
			values = make([]float32, len(in[0].F))
			for i, x := range in[0].F {
				if originalAVX512Unary(threads) {
					values[i] = MKLLogAVX512(x)
				} else {
					values[i] = MKLLog(x)
				}
			}
		} else {
			values = logValues(in[0].F)
		}
		return []Value{{Kind: 1, Shape: append([]int64(nil), in[0].Shape...), F: values}}, nil
	}
	if n.op == "ReduceMean" {
		return graphMeanPair(n, in)
	}
	if n.op == "ReduceSum" {
		return graphIntegerReduce(n, in)
	}
	if n.op == "Range" {
		return graphIntegerRange(n, in)
	}
	if out, handled, err := executeSequenceNode(n, in, original); handled {
		return out, err
	}
	if n.op == "Conv" {
		return graphConv(n, in, original, threads...)
	}
	if n.op == "ConvTranspose" {
		return graphConvTranspose(n, in, original, threads...)
	}
	if n.op == "Where" {
		return graphWhere(n, in)
	}
	if out, handled, err := executeShapeNode(n, in); handled {
		return out, err
	}
	allowed := map[string][]string{"Constant": {"value"}, "Gather": {"axis"}, "Transpose": {"perm"}, "Unsqueeze": {}, "MatMul": {}, "Add": {}, "Sub": {}, "Mul": {}, "Div": {}, "Equal": {}, "Greater": {}, "Less": {}, "GreaterOrEqual": {}, "Clip": {}, "Abs": {}, "Relu": {}, "LeakyRelu": {"alpha"}, "Exp": {}}
	attrs, ok := allowed[n.op]
	if !ok {
		return nil, fmt.Errorf("operation is not yet implemented")
	}
	for name := range n.attrs {
		found := false
		for _, a := range attrs {
			found = found || a == name
		}
		if !found {
			return nil, fmt.Errorf("unsupported attribute %s", name)
		}
	}
	required := 2
	switch n.op {
	case "Constant":
		required = 0
	case "Transpose", "Abs", "Relu", "LeakyRelu", "Exp":
		required = 1
	case "Clip":
		required = 3
	}
	if len(in) != required {
		return nil, fmt.Errorf("input count differs")
	}
	var out Value
	var err error
	switch n.op {
	case "Constant":
		a, ok := n.attrs["value"]
		if !ok || a.tensor == nil {
			return nil, fmt.Errorf("Constant requires a tensor")
		}
		out = *a.tensor
	case "Transpose":
		v := in[0]
		perm := n.attrs["perm"].ints
		if len(perm) == 0 {
			perm = make([]int64, len(v.Shape))
			for i := range perm {
				perm[i] = int64(len(perm) - 1 - i)
			}
		}
		if len(perm) != len(v.Shape) {
			return nil, fmt.Errorf("invalid transpose rank")
		}
		shape := make([]int64, len(perm))
		seen := make([]bool, len(perm))
		for i, p := range perm {
			if p < 0 || p >= int64(len(perm)) || seen[p] {
				return nil, fmt.Errorf("invalid transpose permutation")
			}
			seen[p] = true
			shape[i] = v.Shape[p]
		}
		out, err = emptyValue(v.Kind, shape)
		if err != nil {
			return nil, err
		}
		count, _ := valueCount(shape)
		strides := shapeStrides(v.Shape)
		for index := 0; index < count; index++ {
			offset, remaining := 0, index
			for d := len(shape) - 1; d >= 0; d-- {
				offset += (remaining % int(shape[d])) * strides[perm[d]]
				remaining /= int(shape[d])
			}
			copyElement(&out, index, v, offset)
		}
	case "Unsqueeze":
		v, axes := in[0], in[1]
		if axes.Kind != 7 || len(axes.Shape) != 1 {
			return nil, fmt.Errorf("Unsqueeze axes must be INT64 vector")
		}
		rank := len(v.Shape) + len(axes.I)
		insert := make([]bool, rank)
		for _, a := range axes.I {
			if a < 0 {
				a += int64(rank)
			}
			if a < 0 || a >= int64(rank) || insert[a] {
				return nil, fmt.Errorf("invalid Unsqueeze axes")
			}
			insert[a] = true
		}
		shape := make([]int64, rank)
		d := 0
		for i := range shape {
			if insert[i] {
				shape[i] = 1
			} else {
				shape[i] = v.Shape[d]
				d++
			}
		}
		out = v
		out.Shape = shape
	case "Gather":
		v, indices := in[0], in[1]
		if indices.Kind != 7 || len(v.Shape) == 0 {
			return nil, fmt.Errorf("invalid Gather input")
		}
		axis := n.attrs["axis"].i
		if axis < 0 {
			axis += int64(len(v.Shape))
		}
		if axis < 0 || axis >= int64(len(v.Shape)) {
			return nil, fmt.Errorf("invalid Gather axis")
		}
		a := int(axis)
		shape := append([]int64(nil), v.Shape[:a]...)
		shape = append(shape, indices.Shape...)
		shape = append(shape, v.Shape[a+1:]...)
		out, err = emptyValue(v.Kind, shape)
		if err != nil {
			return nil, err
		}
		outer, _ := valueCount(v.Shape[:a])
		inner, _ := valueCount(v.Shape[a+1:])
		width := int(v.Shape[a])
		at := 0
		for o := 0; o < outer; o++ {
			for _, index := range indices.I {
				if index < 0 {
					index += int64(width)
				}
				if index < 0 || index >= int64(width) {
					return nil, fmt.Errorf("Gather index outside axis")
				}
				base := (o*width + int(index)) * inner
				for k := 0; k < inner; k++ {
					copyElement(&out, at, v, base+k)
					at++
				}
			}
		}
	case "MatMul":
		a, b := in[0], in[1]
		if a.Kind != 1 || b.Kind != 1 || len(a.Shape) < 2 || len(b.Shape) != 2 || a.Shape[len(a.Shape)-1] != b.Shape[0] || b.Shape[0] <= 0 || b.Shape[0] > 256 {
			return nil, fmt.Errorf("unsupported MatMul shapes")
		}
		l := Linear{In: int(b.Shape[0]), Out: int(b.Shape[1]), weight: b.F}
		var values []float32
		var err error
		if original {
			if l.Out == 256 && originalFMAEmbeddings(threads) {
				values, err = l.RunOriginalEmbeddingFMA(a.F, len(a.F)/l.In)
			} else if originalFMAProjections(threads) && originalProjectionFMAShape(l.In, l.Out, len(a.F)/l.In) {
				values, err = l.RunOriginalProjectionFMA(a.F, len(a.F)/l.In)
			} else {
				values, err = l.runOriginalLinear(a.F, len(a.F)/l.In, originalThreadCount(threads))
			}
		} else {
			values, err = l.Run(a.F, len(a.F)/l.In)
		}
		if err != nil {
			return nil, err
		}
		shape := append([]int64(nil), a.Shape...)
		shape[len(shape)-1] = b.Shape[1]
		out = Value{Kind: 1, Shape: shape, F: values}
	case "Add", "Sub", "Mul", "Div", "Equal", "Greater", "Less", "GreaterOrEqual":
		out, err = binaryValue(n.op, in[0], in[1])
		if err != nil {
			return nil, err
		}
	case "Clip", "Abs", "Relu", "LeakyRelu", "Exp":
		out = cloneValue(in[0])
		if n.op == "Abs" && (out.Kind == 6 || out.Kind == 7) {
			for i, x := range out.I {
				if x < 0 {
					out.I[i] = -x
				}
				if out.Kind == 6 {
					out.I[i] = int64(int32(out.I[i]))
				}
			}
			return []Value{out}, nil
		}
		if out.Kind != 1 {
			return nil, fmt.Errorf("unary arithmetic requires FLOAT")
		}
		low, high := float32(math.Inf(-1)), float32(math.Inf(1))
		if n.op == "Clip" {
			for i := 1; i <= 2; i++ {
				if in[i].Kind == 0 {
					continue
				}
				if in[i].Kind != 1 || len(in[i].F) != 1 {
					return nil, fmt.Errorf("Clip requires scalar FLOAT bounds")
				}
				if i == 1 {
					low = in[i].F[0]
				} else {
					high = in[i].F[0]
				}
			}
		}
		for i, x := range out.F {
			switch n.op {
			case "Exp":
				if original {
					if originalAVX512Unary(threads) {
						out.F[i] = MKLExpAVX512(x)
					} else {
						out.F[i] = MKLExp(x)
					}
				} else {
					out.F[i] = Exp(x)
				}
			case "Clip":
				out.F[i] = clamp(x, low, high)
			case "Abs":
				out.F[i] = math.Float32frombits(math.Float32bits(x) & 0x7fffffff)
			case "Relu":
				if x < 0 {
					out.F[i] = 0
				}
			case "LeakyRelu":
				alpha := float32(0.01)
				if a, ok := n.attrs["alpha"]; ok {
					alpha = a.f
				}
				if x < 0 {
					out.F[i] = float32(x * alpha)
				}
			}
		}
	}
	return []Value{out}, nil
}

func shapeStrides(shape []int64) []int {
	strides := make([]int, len(shape))
	n := 1
	for i := len(shape) - 1; i >= 0; i-- {
		strides[i] = n
		n *= int(shape[i])
	}
	return strides
}

func binaryValue(op string, a, b Value) (Value, error) {
	if a.Kind != b.Kind {
		return Value{}, fmt.Errorf("binary input kinds differ")
	}
	rank := max(len(a.Shape), len(b.Shape))
	as, bs := make([]int64, rank), make([]int64, rank)
	for i := range as {
		as[i], bs[i] = 1, 1
	}
	copy(as[rank-len(a.Shape):], a.Shape)
	copy(bs[rank-len(b.Shape):], b.Shape)
	shape := make([]int64, rank)
	for i := range shape {
		if as[i] == bs[i] {
			shape[i] = as[i]
		} else if as[i] == 1 {
			shape[i] = bs[i]
		} else if bs[i] == 1 {
			shape[i] = as[i]
		} else {
			return Value{}, fmt.Errorf("input shapes do not broadcast")
		}
	}
	kind := a.Kind
	if op == "Equal" || op == "Greater" || op == "Less" || op == "GreaterOrEqual" {
		kind = 9
	}
	out, err := emptyValue(kind, shape)
	if err != nil {
		return out, err
	}
	count, _ := valueCount(shape)
	astride, bstride := shapeStrides(as), shapeStrides(bs)
	for index := 0; index < count; index++ {
		ai, bi, remaining := 0, 0, index
		for d := rank - 1; d >= 0; d-- {
			coord := remaining % int(shape[d])
			remaining /= int(shape[d])
			if as[d] != 1 {
				ai += coord * astride[d]
			}
			if bs[d] != 1 {
				bi += coord * bstride[d]
			}
		}
		if op == "Equal" || op == "Greater" || op == "Less" || op == "GreaterOrEqual" {
			switch a.Kind {
			case 1:
				x, y := a.F[ai], b.F[bi]
				switch op {
				case "Equal":
					out.B[index] = x == y
				case "Greater":
					out.B[index] = x > y
				case "GreaterOrEqual":
					out.B[index] = x >= y
				case "Less":
					out.B[index] = x < y
				}
			case 6, 7:
				x, y := a.I[ai], b.I[bi]
				switch op {
				case "Equal":
					out.B[index] = x == y
				case "Greater":
					out.B[index] = x > y
				case "GreaterOrEqual":
					out.B[index] = x >= y
				case "Less":
					out.B[index] = x < y
				}
			case 9:
				if op != "Equal" {
					return out, fmt.Errorf("ordered BOOL comparison unsupported")
				}
				out.B[index] = a.B[ai] == b.B[bi]
			default:
				return out, fmt.Errorf("Equal kind unsupported")
			}
			continue
		}
		if a.Kind != 1 {
			if a.Kind == 6 || a.Kind == 7 {
				switch op {
				case "Add":
					out.I[index] = a.I[ai] + b.I[bi]
				case "Mul":
					out.I[index] = a.I[ai] * b.I[bi]
				case "Sub":
					out.I[index] = a.I[ai] - b.I[bi]
				case "Div":
					if b.I[bi] == 0 {
						return out, fmt.Errorf("integer division by zero")
					}
					out.I[index] = a.I[ai] / b.I[bi]
				}
				if a.Kind == 6 {
					out.I[index] = int64(int32(out.I[index]))
				}
				continue
			}
			return out, fmt.Errorf("binary arithmetic requires FLOAT")
		}
		x, y := a.F[ai], b.F[bi]
		switch op {
		case "Add":
			out.F[index] = float32(x + y)
		case "Sub":
			out.F[index] = float32(x - y)
		case "Mul":
			out.F[index] = float32(x * y)
		case "Div":
			out.F[index] = float32(x / y)
		}
	}
	return out, nil
}
