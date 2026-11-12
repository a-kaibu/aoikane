//go:build amd64

package onnx

const fastConvTile = 24

//go:noescape
func fastConv4x24(y *float32, ystride uintptr, x *float32, xoff *int32, w *float32, bias *float32, steps, wcol, wstep uintptr)

func fastConvKernel(y *float32, ystride uintptr, x *float32, xoff *int32, w *float32, bias *float32, steps, wcol, wstep uintptr) {
	fastConv4x24(y, ystride, x, xoff, w, bias, steps, wcol, wstep)
}
