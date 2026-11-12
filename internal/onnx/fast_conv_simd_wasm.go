//go:build goexperiment.simd && js && wasm

package onnx

import (
	"simd/archsimd"
	"unsafe"
)

const fastConvTile = 8

// fastConvKernel computes four channels by eight frames directly into y.
// runFastStrided validates the geometry and supplies padded inputs, weights,
// and bias; no indexed packing, temporary accumulator, or scalar tail is
// needed here. This kernel is only called for opt-in reordered convolutions.
func fastConvKernel(y *float32, ystride uintptr, x *float32, xoff *int32, w *float32, bias *float32, steps, wcol, wstep uintptr) {
	a0 := archsimd.BroadcastFloat32x4(loadWasmFloat(bias, 0))
	b0 := a0
	a1 := archsimd.BroadcastFloat32x4(loadWasmFloat(bias, 1))
	b1 := a1
	a2 := archsimd.BroadcastFloat32x4(loadWasmFloat(bias, 2))
	b2 := a2
	a3 := archsimd.BroadcastFloat32x4(loadWasmFloat(bias, 3))
	b3 := a3
	for j := uintptr(0); j < steps; j++ {
		o := int(*(*int32)(unsafe.Add(unsafe.Pointer(xoff), j*4)))
		xv := loadWasmVector(x, o)
		xu := loadWasmVector(x, o+4)
		wi := int(j * wstep)
		v0 := archsimd.BroadcastFloat32x4(loadWasmFloat(w, wi))
		v1 := archsimd.BroadcastFloat32x4(loadWasmFloat(w, int(wcol)+wi))
		v2 := archsimd.BroadcastFloat32x4(loadWasmFloat(w, int(2*wcol)+wi))
		v3 := archsimd.BroadcastFloat32x4(loadWasmFloat(w, int(3*wcol)+wi))
		a0 = xv.Mul(v0).Add(a0)
		b0 = xu.Mul(v0).Add(b0)
		a1 = xv.Mul(v1).Add(a1)
		b1 = xu.Mul(v1).Add(b1)
		a2 = xv.Mul(v2).Add(a2)
		b2 = xu.Mul(v2).Add(b2)
		a3 = xv.Mul(v3).Add(a3)
		b3 = xu.Mul(v3).Add(b3)
	}
	a0.StoreArray((*[4]float32)(unsafe.Pointer(y)))
	b0.StoreArray((*[4]float32)(unsafe.Add(unsafe.Pointer(y), 16)))
	a1.StoreArray((*[4]float32)(unsafe.Add(unsafe.Pointer(y), ystride*4)))
	b1.StoreArray((*[4]float32)(unsafe.Add(unsafe.Pointer(y), ystride*4+16)))
	a2.StoreArray((*[4]float32)(unsafe.Add(unsafe.Pointer(y), ystride*8)))
	b2.StoreArray((*[4]float32)(unsafe.Add(unsafe.Pointer(y), ystride*8+16)))
	a3.StoreArray((*[4]float32)(unsafe.Add(unsafe.Pointer(y), ystride*12)))
	b3.StoreArray((*[4]float32)(unsafe.Add(unsafe.Pointer(y), ystride*12+16)))
}
