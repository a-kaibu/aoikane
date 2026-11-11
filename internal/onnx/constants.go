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
	constantFloats = map[constantID]int{}
	packedWeights  = map[packedKey]any{}
)

type packedKey struct {
	weight                weak.Pointer[float32]
	mapped                uintptr // weights in a model mapping, which is never released
	in, out, kernel, kind int
}

// constantKey identifies weights; weak pointers exist only for heap memory.
func constantKey(f []float32) (weak.Pointer[float32], uintptr) {
	p := unsafe.SliceData(f)
	if inMappedRegion(unsafe.Pointer(p)) {
		return weak.Pointer[float32]{}, uintptr(unsafe.Pointer(p))
	}
	return weak.Make(p), 0
}

type constantID struct {
	weak   weak.Pointer[float32]
	mapped uintptr
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
	w, m := constantKey(f)
	constantFloats[constantID{w, m}] = len(f)
}

// sweepConstantsLocked drops entries whose weights were collected.
func sweepConstantsLocked() {
	for k := range constantFloats {
		if k.mapped == 0 && k.weak.Value() == nil {
			delete(constantFloats, k)
		}
	}
	for k := range packedWeights {
		if k.mapped == 0 && k.weight.Value() == nil {
			delete(packedWeights, k)
		}
	}
}

func isConstant(f []float32) bool {
	if len(f) == 0 {
		return false
	}
	w, m := constantKey(f)
	constantMu.Lock()
	n, ok := constantFloats[constantID{w, m}]
	constantMu.Unlock()
	return ok && n == len(f)
}

// cachedPacking returns build() for constant weights, computing it once.
func cachedPacking[T any](weight []float32, in, out, kernel, kind int, build func() T) T {
	if !isConstant(weight) {
		return build()
	}
	w, m := constantKey(weight)
	key := packedKey{w, m, in, out, kernel, kind}
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
