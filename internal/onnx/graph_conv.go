package onnx

import "fmt"

// configureConv is shared by the isolated kernel reader and graph execution,
// so padding and supported kernel geometry have the same validation.
func configureConv(c *Conv1D, attrs map[string]graphAttr) error {
	autoPad, pads := false, false
	for name, a := range attrs {
		switch name {
		case "group":
			if a.i != 1 {
				return fmt.Errorf("only group-one Conv supported")
			}
		case "strides", "dilations", "kernel_shape", "pads":
			v := a.ints
			if name == "pads" {
				pads = true
				if len(v) != 2 || v[0] < 0 || v[1] < 0 || uint64(v[0]) > uint64(^uint(0)>>1) || uint64(v[1]) > uint64(^uint(0)>>1) {
					return fmt.Errorf("invalid Conv pads")
				}
				c.Left, c.Right = int(v[0]), int(v[1])
			} else if name == "kernel_shape" {
				if len(v) != 1 || v[0] != int64(c.Kernel) {
					return fmt.Errorf("unsupported Conv %s", name)
				}
			} else {
				if len(v) != 1 || v[0] < 1 || uint64(v[0]) > uint64(^uint(0)>>1) {
					return fmt.Errorf("invalid Conv %s", name)
				}
				if name == "strides" {
					c.Stride = int(v[0])
				} else {
					c.Dilation = int(v[0])
				}
			}
		case "auto_pad":
			switch a.text {
			case "SAME_UPPER":
				autoPad = true
				c.Left = (c.Kernel - 1) / 2
				c.Right = c.Kernel - 1 - c.Left
			case "VALID":
				autoPad = true
			case "NOTSET":
			default:
				return fmt.Errorf("unsupported Conv auto_pad %s", a.text)
			}
		default:
			return fmt.Errorf("unsupported Conv attribute %s", name)
		}
	}
	if autoPad && pads {
		return fmt.Errorf("explicit pads with auto_pad unsupported")
	}
	if autoPad && (c.Stride > 1 || c.Dilation > 1) {
		return fmt.Errorf("non-unit Conv auto_pad is not implemented")
	}
	return nil
}

func graphConv(n graphNode, in []Value, original bool, threads ...int) ([]Value, error) {
	if len(in) < 2 || len(in) > 3 {
		return nil, fmt.Errorf("Conv input count differs")
	}
	x, w := in[0], in[1]
	if x.Kind != 1 || w.Kind != 1 || len(x.Shape) != 3 || len(w.Shape) != 3 || x.Shape[0] != 1 || x.Shape[1] != w.Shape[1] || w.Shape[0] <= 0 || w.Shape[1] <= 0 || w.Shape[2] <= 0 || (w.Shape[0]%4 != 0 && !(original && w.Shape[0] == 1)) {
		return nil, fmt.Errorf("unsupported Conv input or weight shape")
	}
	c := &Conv1D{In: int(w.Shape[1]), Out: int(w.Shape[0]), Kernel: int(w.Shape[2]), weight: w.F}
	if len(in) == 3 {
		b := in[2]
		if b.Kind != 1 || len(b.Shape) != 1 || b.Shape[0] != w.Shape[0] {
			return nil, fmt.Errorf("Conv bias shape differs")
		}
		c.bias = b.F
	}
	if err := configureConv(c, n.attrs); err != nil {
		return nil, err
	}
	var values []float32
	var err error
	if original {
		if originalFMASmallConv(threads) && c.Kernel >= 1 && c.In <= 320/c.Kernel && c.Out >= 6 && c.Stride <= 1 && c.Dilation <= 1 && (len(x.F) <= 20480 || c.Kernel == 1) {
			values, err = c.RunOriginalSmallConvFMA(x.F, int(x.Shape[2]))
		} else if originalFMASmallConv(threads) && originalBlockedConvShape(c.In, c.Out, c.Kernel) && c.Stride <= 1 && c.Dilation <= 1 && (len(x.F) <= 20480 || c.Kernel == 1) {
			values, err = c.RunOriginalBlockedConvFMA(x.F, int(x.Shape[2]))
		} else {
			values, err = c.runOriginal(x.F, int(x.Shape[2]), originalThreadCount(threads), originalAVX512Convolutions(threads))
		}
	} else {
		values, err = c.Run(x.F, int(x.Shape[2]))
	}
	if err != nil {
		return nil, err
	}
	return []Value{{Kind: 1, Shape: []int64{1, w.Shape[0], int64(len(values) / c.Out)}, F: values}}, nil
}
