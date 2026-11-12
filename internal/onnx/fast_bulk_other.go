//go:build !(js && wasm)

package onnx

func fastConvBulk([]float32, int, []float32, []int32, []float32, []float32, int, int, int) bool {
	return false
}

func fastLeakyBulk([]float32, float32) bool { return false }

func fastGEMMNormalBulk([]float32, []float32, []float32, []float32, int, int, int, int, int, int, int, float32, float32) bool {
	return false
}

func fastProjectionBulk([]float32, []float32, int, int, int, int) ([]float32, bool) {
	return nil, false
}

func fastLSTMGatesBulk([]float32, []float32, []float32, *lstmLayer, int) bool { return false }

func fastGEMVNormalBulk([]float32, []float32, []float32, int, int, int, int, float32) bool {
	return false
}

func fastGatherBulk([]float32, []float32, []int32, int, int) bool { return false }

func fastIndexedConvBulk([]float32, int, []float32, []int32, []int32, []float32, []float32, int, int, int) bool {
	return false
}

func fastBinaryBulk(string, []float32, []float32, []float32) bool { return false }
