//go:build !amd64 && !(goexperiment.simd && js && wasm)

package onnx

const fastConvTile = 24

func fastConvKernel(y *float32, ystride uintptr, x *float32, xoff *int32, w *float32, bias *float32, steps, wcol, wstep uintptr) {
	panic("onnx: fast convolution requires SIMD")
}
