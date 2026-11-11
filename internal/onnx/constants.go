package onnx

import (
	"sync"
	"unsafe"
	"weak"
)

// Initializer FLOAT tensors are immutable for the life of their graph, so
// weight layouts derived from them can be cached by address. Keys are weak
// so that a closed model, its weights and their cached layouts can still be
// collected.
var (
	constantMu     sync.Mutex
	constantFloats = map[weak.Pointer[float32]]int{}
	packedWeights  = map[packedKey]any{}
)

type packedKey struct {
	weight                weak.Pointer[float32]
	in, out, kernel, kind int
}

func registerConstant(f []float32) {
	if len(f) == 0 {
		return
	}
	constantMu.Lock()
	defer constantMu.Unlock()
	if len(constantFloats) >= 1024 && len(constantFloats)&1023 == 0 {
		sweepConstantsLocked()
	}
	constantFloats[weak.Make(unsafe.SliceData(f))] = len(f)
}

// sweepConstantsLocked drops entries whose weights were collected.
func sweepConstantsLocked() {
	for k := range constantFloats {
		if k.Value() == nil {
			delete(constantFloats, k)
		}
	}
	for k := range packedWeights {
		if k.weight.Value() == nil {
			delete(packedWeights, k)
		}
	}
}

func isConstant(f []float32) bool {
	if len(f) == 0 {
		return false
	}
	constantMu.Lock()
	n, ok := constantFloats[weak.Make(unsafe.SliceData(f))]
	constantMu.Unlock()
	return ok && n == len(f)
}

// cachedPacking returns build() for constant weights, computing it once.
func cachedPacking[T any](weight []float32, in, out, kernel, kind int, build func() T) T {
	if !isConstant(weight) {
		return build()
	}
	key := packedKey{weak.Make(unsafe.SliceData(weight)), in, out, kernel, kind}
	constantMu.Lock()
	v, ok := packedWeights[key]
	constantMu.Unlock()
	if ok {
		return v.(T)
	}
	built := build()
	constantMu.Lock()
	defer constantMu.Unlock()
	if v, ok := packedWeights[key]; ok {
		return v.(T)
	}
	packedWeights[key] = built
	return built
}
