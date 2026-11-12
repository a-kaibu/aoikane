//go:build js && wasm

package onnx

import (
	"runtime"
	"syscall/js"
	"unsafe"
)

var (
	convKernel   = js.Global().Get("__aoikaneConvKernel")
	chainKernel  = js.Global().Get("__aoikaneChainKernel")
	leakyKernel  = js.Global().Get("__aoikaneLeakyKernel")
	binaryKernel = js.Global().Get("__aoikaneBinaryKernel")
)

func wasmFloatPointer(f []float32) float64 {
	return float64(uintptr(unsafe.Pointer(unsafe.SliceData(f))))
}

func fastLeakyBulk(x []float32, alpha float32) bool {
	if len(x) < 128 || leakyKernel.Type() != js.TypeFunction {
		return false
	}
	leakyKernel.Invoke(wasmFloatPointer(x), len(x), alpha)
	runtime.KeepAlive(x)
	return true
}

func fastBinaryBulk(op string, out, a, b []float32) bool {
	if len(out) < 128 || (len(a) != 1 && len(a) != len(out)) || (len(b) != 1 && len(b) != len(out)) || binaryKernel.Type() != js.TypeFunction {
		return false
	}
	code := 0
	switch op {
	case "Add":
		code = 1
	case "Sub":
		code = 2
	case "Mul":
		code = 3
	case "Div":
		code = 4
	default:
		return false
	}
	binaryKernel.Invoke(wasmFloatPointer(out), wasmFloatPointer(a), wasmFloatPointer(b), len(out), len(a), len(b), code)
	runtime.KeepAlive(out)
	runtime.KeepAlive(a)
	runtime.KeepAlive(b)
	return true
}

// The small optional module imports the Go module's memory. Calling once per
// convolution avoids copying tensors or crossing the JS boundary per tile.
func fastConvBulk(y []float32, frames int, x []float32, xoff []int32, w, bias []float32, steps, wcol, wstep int) bool {
	fn := convKernel
	if fn.Type() != js.TypeFunction {
		return false
	}
	ptr := func(p *float32) float64 { return float64(uintptr(unsafe.Pointer(p))) }
	ok := fn.Invoke(ptr(unsafe.SliceData(y)), frames, len(bias), ptr(unsafe.SliceData(x)), float64(uintptr(unsafe.Pointer(unsafe.SliceData(xoff)))), ptr(unsafe.SliceData(w)), ptr(unsafe.SliceData(bias)), steps, wcol, wstep).Bool()
	runtime.KeepAlive(y)
	runtime.KeepAlive(x)
	runtime.KeepAlive(xoff)
	runtime.KeepAlive(w)
	runtime.KeepAlive(bias)
	return ok
}

func fastChainsBulk(acc []float32, rows int, x []float32, base int, xoff []int32, w [chainCols][]float32, step int, woff []int32, fused bool) bool {
	fn := chainKernel
	if fn.Type() != js.TypeFunction {
		return false
	}
	ptr := func(f []float32) float64 { return float64(uintptr(unsafe.Pointer(unsafe.SliceData(f)))) }
	fuse := 0
	if fused {
		fuse = 1
	}
	fn.Invoke(ptr(acc), rows, ptr(x), base, float64(uintptr(unsafe.Pointer(unsafe.SliceData(xoff)))), ptr(w[0]), ptr(w[1]), ptr(w[2]), ptr(w[3]), len(xoff), step, float64(uintptr(unsafe.Pointer(unsafe.SliceData(woff)))), fuse)
	runtime.KeepAlive(acc)
	runtime.KeepAlive(x)
	runtime.KeepAlive(xoff)
	runtime.KeepAlive(w)
	runtime.KeepAlive(woff)
	return true
}
