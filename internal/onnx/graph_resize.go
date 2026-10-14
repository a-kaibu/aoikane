package onnx

import (
	"fmt"
	"math"
)

// The vocoder uses integer nearest-neighbor expansion of [N,C,T] tensors.
// Other interpolation modes and noninteger spatial ratios are not implemented.
func graphResizeNearest1D(n graphNode, in []Value) ([]Value, error) {
	for name := range n.attrs {
		switch name {
		case "mode", "nearest_mode", "coordinate_transformation_mode", "cubic_coeff_a":
		default:
			return nil, fmt.Errorf("unsupported Resize attribute %s", name)
		}
	}
	if n.attrs["mode"].text != "nearest" || n.attrs["nearest_mode"].text != "floor" || n.attrs["coordinate_transformation_mode"].text != "asymmetric" {
		return nil, fmt.Errorf("Resize requires asymmetric nearest/floor mode")
	}
	if (len(in) != 3 && len(in) != 4) || in[0].Kind != 1 || len(in[0].Shape) != 3 {
		return nil, fmt.Errorf("Resize requires FLOAT [N,C,T] and sizes/scales")
	}
	for _, v := range in[1:2] {
		if v.Kind != 0 && (v.Kind != 1 || len(v.F) != 0) {
			return nil, fmt.Errorf("Resize ROI must be empty")
		}
	}
	x := in[0]
	var shape []int64
	if len(in) == 4 && in[3].Kind != 0 {
		if in[3].Kind != 7 || len(in[3].Shape) != 1 || len(in[3].I) != 3 || (in[2].Kind != 0 && (in[2].Kind != 1 || len(in[2].F) != 0)) {
			return nil, fmt.Errorf("invalid Resize sizes/scales")
		}
		shape = in[3].I
	} else {
		scales := in[2]
		if scales.Kind != 1 || len(scales.Shape) != 1 || len(scales.F) != 3 || scales.F[0] != 1 || scales.F[1] != 1 {
			return nil, fmt.Errorf("Resize requires spatial-only FLOAT scales")
		}
		ratio := float64(scales.F[2])
		if ratio < 1 || ratio > 32768 || math.Trunc(ratio) != ratio || x.Shape[2] < 1 || x.Shape[2] > math.MaxInt64/int64(ratio) {
			return nil, fmt.Errorf("Resize requires integer expansion scales")
		}
		shape = []int64{x.Shape[0], x.Shape[1], x.Shape[2] * int64(ratio)}
	}
	if shape[0] != x.Shape[0] || shape[1] != x.Shape[1] || x.Shape[2] < 1 || shape[2] < x.Shape[2] || shape[2]%x.Shape[2] != 0 {
		return nil, fmt.Errorf("Resize requires integer expansion of the time axis only")
	}
	y, err := emptyValue(1, shape)
	if err != nil {
		return nil, err
	}
	width, expanded, ratio := int(x.Shape[2]), int(shape[2]), int(shape[2]/x.Shape[2])
	for row := 0; row < len(x.F)/width; row++ {
		for i := 0; i < expanded; i++ {
			y.F[row*expanded+i] = x.F[row*width+i/ratio]
		}
	}
	return []Value{y}, nil
}
