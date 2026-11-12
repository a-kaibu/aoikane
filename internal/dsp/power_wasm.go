//go:build js && wasm

package dsp

import (
	"runtime"
	"syscall/js"
	"unsafe"
)

var powerKernel = js.Global().Get("__aoikanePowerKernel")

func windowPowerSums(wave, window []float32) ([]float32, bool) {
	if powerKernel.Type() != js.TypeFunction || len(window) != 2048 || len(wave)%240 != 0 {
		return nil, false
	}
	// The original window has 904 leading samples. One padded copy eliminates
	// millions of per-sample range checks without changing the reduction tree.
	padded := make([]float32, len(wave)+2048)
	copy(padded[904:], wave)
	out := make([]float32, len(wave)/240)
	ptr := func(f []float32) float64 { return float64(uintptr(unsafe.Pointer(unsafe.SliceData(f)))) }
	powerKernel.Invoke(ptr(out), ptr(padded), ptr(window), len(out))
	runtime.KeepAlive(out)
	runtime.KeepAlive(padded)
	runtime.KeepAlive(window)
	return out, true
}
