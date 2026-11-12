//go:build !(js && wasm)

package onnx

func fastConvBulk([]float32, int, []float32, []int32, []float32, []float32, int, int, int) bool {
	return false
}

func fastLeakyBulk([]float32, float32) bool { return false }

func fastBinaryBulk(string, []float32, []float32, []float32) bool { return false }
