//go:build !(goexperiment.simd && js && wasm)

package onnx

func fmaChainsSIMD([]float32, int, []float32, int, []int32, [chainCols][]float32, int, bool) bool {
	return false
}

func fmaChainColSIMD([]float32, []float32, int, []int32, []float32, int) bool {
	return false
}

func fmaChainsIndexedSIMD([]float32, int, []float32, int, []int32, [chainCols][]float32, []int32) bool {
	return false
}
