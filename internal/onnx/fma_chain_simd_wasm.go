//go:build goexperiment.simd && js && wasm

package onnx

import (
	"simd/archsimd"
	"unsafe"
)

// fmaChainsSIMD evaluates four independent accumulation lanes per WebAssembly
// SIMD instruction. Lanes remain independent, so each lane still accumulates
// its reduction steps in the scalar order. The fused and unfused forms both
// use multiply-then-add because standard WebAssembly SIMD has no FMA; callers
// only select this path when reordered kernels are enabled.
func fmaChainsSIMD(acc []float32, rows int, x []float32, xbase int, xoff []int32, w [chainCols][]float32, wstep int, fused bool) bool {
	if rows < 4 || len(acc) < chainCols*rows || len(xoff) == 0 {
		return false
	}
	lo, hi := int64(xoff[0]), int64(xoff[0])
	for _, o := range xoff {
		v := int64(o)
		lo = min(lo, v)
		hi = max(hi, v)
	}
	if int64(xbase)+lo < 0 || int64(xbase)+hi+int64(rows) > int64(len(x)) {
		return false
	}
	lanes := rows &^ 3
	for c := 0; c < chainCols; c++ {
		wc := w[c]
		dst := acc[c*rows : (c+1)*rows]
		for r := 0; r < lanes; r += 4 {
			a := archsimd.LoadFloat32x4(dst[r:])
			for j, o := range xoff {
				base := xbase + int(o) + r
				xv := archsimd.LoadFloat32x4Array((*[4]float32)(unsafe.Pointer(&x[base])))
				wv := archsimd.BroadcastFloat32x4(wc[j*wstep])
				a = xv.Mul(wv).Add(a)
			}
			a.StoreArray((*[4]float32)(unsafe.Pointer(&dst[r])))
		}
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
	lo, hi := int64(xoff[0]), int64(xoff[0])
	for _, o := range xoff {
		v := int64(o)
		lo = min(lo, v)
		hi = max(hi, v)
	}
	if int64(xbase)+lo < 0 || int64(xbase)+hi+int64(len(acc)) > int64(len(x)) {
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
	if rows < 4 || len(acc) < chainCols*rows || len(xoff) == 0 || len(woff) != len(xoff) {
		return false
	}
	lo, hi := int64(xoff[0]), int64(xoff[0])
	for _, o := range xoff {
		v := int64(o)
		lo = min(lo, v)
		hi = max(hi, v)
	}
	if int64(xbase)+lo < 0 || int64(xbase)+hi+int64(rows) > int64(len(x)) {
		return false
	}
	lanes := rows &^ 3
	for c := 0; c < chainCols; c++ {
		wc := w[c]
		dst := acc[c*rows : (c+1)*rows]
		for r := 0; r < lanes; r += 4 {
			a := archsimd.LoadFloat32x4(dst[r:])
			for j, o := range xoff {
				base := xbase + int(o) + r
				xv := archsimd.LoadFloat32x4Array((*[4]float32)(unsafe.Pointer(&x[base])))
				wv := archsimd.BroadcastFloat32x4(wc[woff[j]])
				a = xv.Mul(wv).Add(a)
			}
			a.StoreArray((*[4]float32)(unsafe.Pointer(&dst[r])))
		}
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
