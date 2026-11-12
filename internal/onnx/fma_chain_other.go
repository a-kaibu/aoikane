//go:build !amd64

package onnx

import "runtime"

var useAVX2FMA = false

// useChainedKernels selects the lane-parallel reduction order even without
// AVX2 assembly. WebAssembly uses the pure Go fallback in fmaChains, which
// keeps the same scalar accumulation order per lane.
var useChainedKernels = runtime.GOARCH == "wasm"

func fmaChainsAsm(acc []float32, rows int, x []float32, xbase int, xoff []int32, w [chainCols][]float32, wstep int, fused bool) {
	panic("onnx: no assembly FMA chains")
}

func fmaChainColAsm(acc []float32, x []float32, xbase int, xoff []int32, w []float32, wstep int) {
	panic("onnx: no assembly FMA chains")
}

func fmaChainsIndexedAsm(acc []float32, rows int, x []float32, xbase int, xoff []int32, w [chainCols][]float32, woff []int32) {
	panic("onnx: no assembly FMA chains")
}
