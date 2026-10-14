package onnx

import "fmt"

// graphAveragePool implements the unit-stride 1D NCHW average pools in FeatNet.
func graphAveragePool(n graphNode, in []Value) ([]Value, error) {
	if len(in) != 1 || in[0].Kind != 1 || len(in[0].Shape) != 3 || in[0].Shape[0] != 1 {
		return nil, fmt.Errorf("AveragePool requires batch-one FLOAT NCHW")
	}
	for name, a := range n.attrs {
		switch name {
		case "kernel_shape", "pads":
		case "ceil_mode", "count_include_pad":
			if a.i != 0 {
				return nil, fmt.Errorf("unsupported AveragePool flag %s", name)
			}
		case "strides":
			if len(a.ints) != 1 || a.ints[0] != 1 {
				return nil, fmt.Errorf("AveragePool requires unit stride")
			}
		default:
			return nil, fmt.Errorf("unsupported AveragePool attribute %s", name)
		}
	}
	kernel, pads := n.attrs["kernel_shape"].ints, n.attrs["pads"].ints
	if len(kernel) != 1 || kernel[0] <= 0 || len(pads) != 2 || pads[0] < 0 || pads[1] < 0 || pads[0] >= kernel[0] || pads[1] >= kernel[0] {
		return nil, fmt.Errorf("invalid AveragePool kernel or pads")
	}
	v := in[0]
	channels, width := v.Shape[1], v.Shape[2]
	limit := int64(^uint(0) >> 1)
	if pads[0] > limit-width || pads[1] > limit-width-pads[0] {
		return nil, fmt.Errorf("AveragePool padding overflows")
	}
	length := width + pads[0] + pads[1] - kernel[0] + 1
	if width <= 0 || length <= 0 {
		return nil, fmt.Errorf("AveragePool requires nonempty input and output")
	}
	out, err := emptyValue(1, []int64{1, channels, length})
	if err != nil {
		return nil, err
	}
	for c := int64(0); c < channels; c++ {
		for j := int64(0); j < length; j++ {
			start, end := max(j-pads[0], int64(0)), min(j-pads[0]+kernel[0], width)
			if end <= start {
				return nil, fmt.Errorf("AveragePool window has no input")
			}
			var sum float32
			for k := start; k < end; k++ {
				sum = float32(sum + v.F[c*width+k])
			}
			out.F[c*length+j] = float32(sum / float32(end-start))
		}
	}
	return []Value{out}, nil
}
