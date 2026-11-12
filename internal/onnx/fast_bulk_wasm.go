//go:build js && wasm

package onnx

import (
	"runtime"
	"syscall/js"
	"unsafe"
)

var (
	gemmNormalKernel  = js.Global().Get("__aoikaneGEMMNormalKernel")
	projectionKernel  = js.Global().Get("__aoikaneProjectionKernel")
	lstmGatesKernel   = js.Global().Get("__aoikaneLSTMGatesKernel")
	gemvNormalKernel  = js.Global().Get("__aoikaneGEMVNormalKernel")
	convKernel        = js.Global().Get("__aoikaneConvKernel")
	chainKernel       = js.Global().Get("__aoikaneChainKernel")
	leakyKernel       = js.Global().Get("__aoikaneLeakyKernel")
	binaryKernel      = js.Global().Get("__aoikaneBinaryKernel")
	gatherKernel      = js.Global().Get("__aoikaneGatherKernel")
	indexedConvKernel = js.Global().Get("__aoikaneIndexedConvKernel")
	orderedConvKernel = js.Global().Get("__aoikaneOrderedConvKernel")
)

func fastGEMMNormalBulk(y, a, b, initial []float32, m, n, k, lda, ldb, ldc, align int, alpha, beta float32) bool {
	if gemmNormalKernel.Type() != js.TypeFunction {
		return false
	}
	gemmNormalKernel.Invoke(wasmFloatPointer(y), wasmFloatPointer(a), wasmFloatPointer(b), wasmFloatPointer(initial), m, n, k, lda, ldb, ldc, align, alpha, beta)
	runtime.KeepAlive(y)
	runtime.KeepAlive(a)
	runtime.KeepAlive(b)
	runtime.KeepAlive(initial)
	return true
}

func fastProjectionBulk(input, w []float32, rows, inner, outputs, threads int) ([]float32, bool) {
	if projectionKernel.Type() != js.TypeFunction || !(outputs == 1 && inner == 64 || outputs == 60 && inner == 256 && rows >= 60) {
		return nil, false
	}
	y := make([]float32, rows*outputs)
	if outputs == 1 {
		projectionKernel.Invoke(wasmFloatPointer(y), wasmFloatPointer(input), wasmFloatPointer(w), inner, outputs, 0, rows)
	} else {
		for _, part := range originalMKLPartitions(rows, threads) {
			projectionKernel.Invoke(wasmFloatPointer(y), wasmFloatPointer(input), wasmFloatPointer(w), inner, outputs, part[0], part[1])
		}
	}
	runtime.KeepAlive(y)
	runtime.KeepAlive(input)
	runtime.KeepAlive(w)
	return y, true
}

func fastLSTMGatesBulk(out, input, state []float32, l *lstmLayer, direction int) bool {
	if lstmGatesKernel.Type() != js.TypeFunction || !FastMath() || l.packing == nil {
		return false
	}
	p := l.packing
	p.once.Do(func() {
		gates := 4 * l.hidden
		p.w = make([]float32, len(l.w))
		p.r = make([]float32, len(l.r))
		for d := 0; d < 2; d++ {
			for g := 0; g < gates; g++ {
				for k := 0; k < l.width; k++ {
					p.w[(d*l.width+k)*gates+g] = l.w[(d*gates+g)*l.width+k]
				}
				for k := 0; k < l.hidden; k++ {
					p.r[(d*l.hidden+k)*gates+g] = l.r[(d*gates+g)*l.hidden+k]
				}
			}
		}
	})
	g := 4 * l.hidden
	lstmGatesKernel.Invoke(wasmFloatPointer(out), wasmFloatPointer(input), wasmFloatPointer(state), wasmFloatPointer(p.w[direction*l.width*g:]), wasmFloatPointer(p.r[direction*l.hidden*g:]), wasmFloatPointer(l.bias[direction*g:]), l.width, l.hidden)
	runtime.KeepAlive(l)
	runtime.KeepAlive(out)
	runtime.KeepAlive(input)
	runtime.KeepAlive(state)
	return true
}

func fastGEMVNormalBulk(y, a, x []float32, m, n, lda, prefix int, alpha float32) bool {
	if gemvNormalKernel.Type() != js.TypeFunction {
		return false
	}
	gemvNormalKernel.Invoke(wasmFloatPointer(y), wasmFloatPointer(a), wasmFloatPointer(x), m, n, lda, prefix, alpha)
	runtime.KeepAlive(y)
	runtime.KeepAlive(a)
	runtime.KeepAlive(x)
	return true
}

// All offsets and spans are constructed from the already validated complete
// convolution in runOriginalDNNLPadded. Cross JS once, not once per 4-column
// group, and evaluate the last partial span with four rather than eight lanes.
func fastIndexedConvBulk(y []float32, frames int, x []float32, xoff, woff []int32, w, bias []float32, channels, wcol, kernel int) bool {
	if indexedConvKernel.Type() != js.TypeFunction || channels%4 != 0 || frames < 8 {
		return false
	}
	ptr := func(p []int32) float64 { return float64(uintptr(unsafe.Pointer(unsafe.SliceData(p)))) }
	if orderedConvKernel.Type() == js.TypeFunction && isConstant(w) {
		packed := cachedPacking(w, len(xoff), channels, kernel, 3, func() []float32 {
			p := make([]float32, channels*len(xoff))
			for c := 0; c < channels; c++ {
				for j, o := range woff {
					p[c*len(xoff)+j] = w[c*wcol+int(o)]
				}
			}
			return p
		})
		orderedConvKernel.Invoke(wasmFloatPointer(y), frames, channels, wasmFloatPointer(x), ptr(xoff), wasmFloatPointer(packed), wasmFloatPointer(bias), len(xoff))
		runtime.KeepAlive(y)
		runtime.KeepAlive(x)
		runtime.KeepAlive(xoff)
		runtime.KeepAlive(w)
		runtime.KeepAlive(packed)
		runtime.KeepAlive(bias)
		return true
	}
	indexedConvKernel.Invoke(wasmFloatPointer(y), frames, channels, wasmFloatPointer(x), ptr(xoff), ptr(woff), wasmFloatPointer(w), wasmFloatPointer(bias), len(xoff), wcol)
	runtime.KeepAlive(y)
	runtime.KeepAlive(x)
	runtime.KeepAlive(xoff)
	runtime.KeepAlive(woff)
	runtime.KeepAlive(w)
	runtime.KeepAlive(bias)
	return true
}

func fastGatherBulk(y, x []float32, positions []int32, channels, frames int) bool {
	if gatherKernel.Type() != js.TypeFunction {
		return false
	}
	gatherKernel.Invoke(wasmFloatPointer(y), wasmFloatPointer(x), float64(uintptr(unsafe.Pointer(unsafe.SliceData(positions)))), channels, frames)
	runtime.KeepAlive(y)
	runtime.KeepAlive(x)
	runtime.KeepAlive(positions)
	return true
}

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
