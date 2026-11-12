//go:build goexperiment.simd && js && wasm

package onnx

import (
	"simd/archsimd"
	"unsafe"
)

const wasmSIMDAvailable = true

// Bounds are checked once per kernel, rather than once per multiply-add.
// Callers must validate every offset before using these unchecked loads.
func loadWasmFloat(p *float32, index int) float32 {
	return *(*float32)(unsafe.Add(unsafe.Pointer(p), index*4))
}

func loadWasmVector(p *float32, index int) archsimd.Float32x4 {
	return archsimd.LoadFloat32x4Array((*[4]float32)(unsafe.Add(unsafe.Pointer(p), index*4)))
}

func validWasmWeights(w [chainCols][]float32, steps, step int) bool {
	if step < 0 {
		return false
	}
	for _, wc := range w {
		if len(wc) == 0 || (step > 0 && steps-1 > (len(wc)-1)/step) {
			return false
		}
	}
	return true
}

func validWasmInput(size, rows, base int, offsets []int32) bool {
	lo, hi := int64(offsets[0]), int64(offsets[0])
	for _, o := range offsets {
		lo = min(lo, int64(o))
		hi = max(hi, int64(o))
	}
	// Compare base against the bounds instead of adding it to offsets:
	// malformed bases near MaxInt must not wrap around the range check.
	return int64(base) >= -lo && int64(base) <= int64(size)-int64(rows)-hi
}

// fmaChainsSIMD tiles four output columns by eight rows: two input vectors
// and four broadcast weights feed eight independent accumulators. Sharing
// loads and validating ranges once keeps indexing out of the hot loop.
// Each lane retains its scalar reduction order. Both forms use multiply-then-
// add because standard WebAssembly SIMD has no FMA; callers only select this
// path when reordered kernels are enabled. Scalar tails retain fused math.
func fmaChainsSIMD(acc []float32, rows int, x []float32, xbase int, xoff []int32, w [chainCols][]float32, wstep int, fused bool) bool {
	if rows < 4 || rows > len(acc)/chainCols || len(xoff) == 0 || !validWasmWeights(w, len(xoff), wstep) {
		return false
	}
	if !validWasmInput(len(x), rows, xbase, xoff) {
		return false
	}
	lanes := rows &^ 3
	bulk := fastChainsBulk(acc, rows, x, xbase, xoff, w, wstep, nil, fused)
	xp := unsafe.SliceData(x)
	w0, w1, w2, w3 := unsafe.SliceData(w[0]), unsafe.SliceData(w[1]), unsafe.SliceData(w[2]), unsafe.SliceData(w[3])
	wide := lanes &^ 7
	for r := 0; r < wide && !bulk; r += 8 {
		a0 := archsimd.LoadFloat32x4(acc[r:])
		b0 := archsimd.LoadFloat32x4(acc[r+4:])
		a1 := archsimd.LoadFloat32x4(acc[rows+r:])
		b1 := archsimd.LoadFloat32x4(acc[rows+r+4:])
		a2 := archsimd.LoadFloat32x4(acc[2*rows+r:])
		b2 := archsimd.LoadFloat32x4(acc[2*rows+r+4:])
		a3 := archsimd.LoadFloat32x4(acc[3*rows+r:])
		b3 := archsimd.LoadFloat32x4(acc[3*rows+r+4:])
		for j, o := range xoff {
			base := xbase + int(o) + r
			xv := loadWasmVector(xp, base)
			xu := loadWasmVector(xp, base+4)
			wi := j * wstep
			v0 := archsimd.BroadcastFloat32x4(loadWasmFloat(w0, wi))
			v1 := archsimd.BroadcastFloat32x4(loadWasmFloat(w1, wi))
			v2 := archsimd.BroadcastFloat32x4(loadWasmFloat(w2, wi))
			v3 := archsimd.BroadcastFloat32x4(loadWasmFloat(w3, wi))
			a0 = xv.Mul(v0).Add(a0)
			b0 = xu.Mul(v0).Add(b0)
			a1 = xv.Mul(v1).Add(a1)
			b1 = xu.Mul(v1).Add(b1)
			a2 = xv.Mul(v2).Add(a2)
			b2 = xu.Mul(v2).Add(b2)
			a3 = xv.Mul(v3).Add(a3)
			b3 = xu.Mul(v3).Add(b3)
		}
		a0.Store(acc[r:])
		b0.Store(acc[r+4:])
		a1.Store(acc[rows+r:])
		b1.Store(acc[rows+r+4:])
		a2.Store(acc[2*rows+r:])
		b2.Store(acc[2*rows+r+4:])
		a3.Store(acc[3*rows+r:])
		b3.Store(acc[3*rows+r+4:])
	}
	for r := wide; r < lanes && !bulk; r += 4 {
		a0 := archsimd.LoadFloat32x4(acc[r:])
		a1 := archsimd.LoadFloat32x4(acc[rows+r:])
		a2 := archsimd.LoadFloat32x4(acc[2*rows+r:])
		a3 := archsimd.LoadFloat32x4(acc[3*rows+r:])
		for j, o := range xoff {
			base := xbase + int(o) + r
			xv := loadWasmVector(xp, base)
			wi := j * wstep
			a0 = xv.Mul(archsimd.BroadcastFloat32x4(loadWasmFloat(w0, wi))).Add(a0)
			a1 = xv.Mul(archsimd.BroadcastFloat32x4(loadWasmFloat(w1, wi))).Add(a1)
			a2 = xv.Mul(archsimd.BroadcastFloat32x4(loadWasmFloat(w2, wi))).Add(a2)
			a3 = xv.Mul(archsimd.BroadcastFloat32x4(loadWasmFloat(w3, wi))).Add(a3)
		}
		a0.Store(acc[r:])
		a1.Store(acc[rows+r:])
		a2.Store(acc[2*rows+r:])
		a3.Store(acc[3*rows+r:])
	}
	for c := 0; c < chainCols; c++ {
		wc := w[c]
		dst := acc[c*rows : (c+1)*rows]
		for r := lanes; r < rows; r++ {
			a := dst[r]
			for j, o := range xoff {
				v := x[xbase+int(o)+r]
				if fused {
					a = fma32(v, wc[j*wstep], a)
				} else {
					a = float32(a + float32(v*wc[j*wstep]))
				}
			}
			dst[r] = a
		}
	}
	return true
}

// fmaChainColSIMD is the single-column form of fmaChainsSIMD.
func fmaChainColSIMD(acc []float32, x []float32, xbase int, xoff []int32, w []float32, wstep int) bool {
	if len(acc) < 4 || len(xoff) == 0 {
		return false
	}
	if !validWasmInput(len(x), len(acc), xbase, xoff) {
		return false
	}
	lanes := len(acc) &^ 3
	for r := 0; r < lanes; r += 4 {
		a := archsimd.LoadFloat32x4(acc[r:])
		for j, o := range xoff {
			base := xbase + int(o) + r
			xv := archsimd.LoadFloat32x4Array((*[4]float32)(unsafe.Pointer(&x[base])))
			wv := archsimd.BroadcastFloat32x4(w[j*wstep])
			a = xv.Mul(wv).Add(a)
		}
		a.StoreArray((*[4]float32)(unsafe.Pointer(&acc[r])))
	}
	for r := lanes; r < len(acc); r++ {
		a := acc[r]
		for j, o := range xoff {
			a = fma32(x[xbase+int(o)+r], w[j*wstep], a)
		}
		acc[r] = a
	}
	return true
}

// fmaChainsIndexedSIMD is fmaChainsSIMD with per-step weight offsets.
func fmaChainsIndexedSIMD(acc []float32, rows int, x []float32, xbase int, xoff []int32, w [chainCols][]float32, woff []int32) bool {
	if rows < 4 || rows > len(acc)/chainCols || len(xoff) == 0 || len(woff) != len(xoff) {
		return false
	}
	for _, offset := range woff {
		for _, wc := range w {
			if offset < 0 || int(offset) >= len(wc) {
				return false
			}
		}
	}
	if !validWasmInput(len(x), rows, xbase, xoff) {
		return false
	}
	lanes := rows &^ 3
	bulk := fastChainsBulk(acc, rows, x, xbase, xoff, w, 0, woff, true)
	xp := unsafe.SliceData(x)
	w0, w1, w2, w3 := unsafe.SliceData(w[0]), unsafe.SliceData(w[1]), unsafe.SliceData(w[2]), unsafe.SliceData(w[3])
	wide := lanes &^ 7
	for r := 0; r < wide && !bulk; r += 8 {
		a0 := archsimd.LoadFloat32x4(acc[r:])
		b0 := archsimd.LoadFloat32x4(acc[r+4:])
		a1 := archsimd.LoadFloat32x4(acc[rows+r:])
		b1 := archsimd.LoadFloat32x4(acc[rows+r+4:])
		a2 := archsimd.LoadFloat32x4(acc[2*rows+r:])
		b2 := archsimd.LoadFloat32x4(acc[2*rows+r+4:])
		a3 := archsimd.LoadFloat32x4(acc[3*rows+r:])
		b3 := archsimd.LoadFloat32x4(acc[3*rows+r+4:])
		for j, o := range xoff {
			base := xbase + int(o) + r
			xv := loadWasmVector(xp, base)
			xu := loadWasmVector(xp, base+4)
			wi := int(woff[j])
			v0 := archsimd.BroadcastFloat32x4(loadWasmFloat(w0, wi))
			v1 := archsimd.BroadcastFloat32x4(loadWasmFloat(w1, wi))
			v2 := archsimd.BroadcastFloat32x4(loadWasmFloat(w2, wi))
			v3 := archsimd.BroadcastFloat32x4(loadWasmFloat(w3, wi))
			a0 = xv.Mul(v0).Add(a0)
			b0 = xu.Mul(v0).Add(b0)
			a1 = xv.Mul(v1).Add(a1)
			b1 = xu.Mul(v1).Add(b1)
			a2 = xv.Mul(v2).Add(a2)
			b2 = xu.Mul(v2).Add(b2)
			a3 = xv.Mul(v3).Add(a3)
			b3 = xu.Mul(v3).Add(b3)
		}
		a0.Store(acc[r:])
		b0.Store(acc[r+4:])
		a1.Store(acc[rows+r:])
		b1.Store(acc[rows+r+4:])
		a2.Store(acc[2*rows+r:])
		b2.Store(acc[2*rows+r+4:])
		a3.Store(acc[3*rows+r:])
		b3.Store(acc[3*rows+r+4:])
	}
	for r := wide; r < lanes && !bulk; r += 4 {
		a0 := archsimd.LoadFloat32x4(acc[r:])
		a1 := archsimd.LoadFloat32x4(acc[rows+r:])
		a2 := archsimd.LoadFloat32x4(acc[2*rows+r:])
		a3 := archsimd.LoadFloat32x4(acc[3*rows+r:])
		for j, o := range xoff {
			base := xbase + int(o) + r
			xv := loadWasmVector(xp, base)
			wi := int(woff[j])
			a0 = xv.Mul(archsimd.BroadcastFloat32x4(loadWasmFloat(w0, wi))).Add(a0)
			a1 = xv.Mul(archsimd.BroadcastFloat32x4(loadWasmFloat(w1, wi))).Add(a1)
			a2 = xv.Mul(archsimd.BroadcastFloat32x4(loadWasmFloat(w2, wi))).Add(a2)
			a3 = xv.Mul(archsimd.BroadcastFloat32x4(loadWasmFloat(w3, wi))).Add(a3)
		}
		a0.Store(acc[r:])
		a1.Store(acc[rows+r:])
		a2.Store(acc[2*rows+r:])
		a3.Store(acc[3*rows+r:])
	}
	for c := 0; c < chainCols; c++ {
		wc := w[c]
		dst := acc[c*rows : (c+1)*rows]
		for r := lanes; r < rows; r++ {
			a := dst[r]
			for j, o := range xoff {
				a = fma32(x[xbase+int(o)+r], wc[woff[j]], a)
			}
			dst[r] = a
		}
	}
	return true
}
