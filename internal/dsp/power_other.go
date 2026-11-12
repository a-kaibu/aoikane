//go:build !(js && wasm)

package dsp

func windowPowerSums([]float32, []float32) ([]float32, bool) { return nil, false }
