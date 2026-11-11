//go:build amd64

package onnx

//go:noescape
func fastConv4x24(y *float32, ystride uintptr, x *float32, xoff *int32, w *float32, bias *float32, steps, wcol, wstep uintptr)
