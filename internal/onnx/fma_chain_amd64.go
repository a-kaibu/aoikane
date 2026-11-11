//go:build amd64

package onnx

import "unsafe"

var useAVX2FMA = detectAVX2FMA()

func cpuidex(leaf, sub uint32) (a, b, c, d uint32)
func xgetbv0() (eax, edx uint32)

func detectAVX2FMA() bool {
	max, _, _, _ := cpuidex(0, 0)
	if max < 7 {
		return false
	}
	_, _, c1, _ := cpuidex(1, 0)
	const fma, osxsave, avx = 1 << 12, 1 << 27, 1 << 28
	if c1&fma == 0 || c1&osxsave == 0 || c1&avx == 0 {
		return false
	}
	if eax, _ := xgetbv0(); eax&6 != 6 { // XMM and YMM state enabled by the OS
		return false
	}
	_, b7, _, _ := cpuidex(7, 0)
	return b7&(1<<5) != 0 // AVX2
}

//go:noescape
func fmaChain4x16(acc, x *float32, xoff *int32, w0, w1, w2, w3 *float32, wstep, steps uintptr)

//go:noescape
func fmaChain4x8(acc, x *float32, xoff *int32, w0, w1, w2, w3 *float32, wstep, steps uintptr)

//go:noescape
func mulAddChain4x16(acc, x *float32, xoff *int32, w0, w1, w2, w3 *float32, wstep, steps uintptr)

//go:noescape
func mulAddChain4x8(acc, x *float32, xoff *int32, w0, w1, w2, w3 *float32, wstep, steps uintptr)

// fmaChainsAsm evaluates every lane of every column. rows is a multiple of
// 8. Bounds are checked once here so the assembly never reads outside x or w.
func fmaChainsAsm(acc []float32, rows int, x []float32, xbase int, xoff []int32, w [chainCols][]float32, wstep int, fused bool) {
	steps := len(xoff)
	lo, hi := int(xoff[0]), int(xoff[0])
	for _, o := range xoff {
		lo, hi = min(lo, int(o)), max(hi, int(o))
	}
	if xbase+lo < 0 || xbase+hi+rows > len(x) || len(acc) < chainCols*rows {
		panic("onnx: FMA chain input out of range")
	}
	for c := range w {
		if (steps-1)*wstep >= len(w[c]) {
			panic("onnx: FMA chain weight out of range")
		}
	}
	f16, f8 := mulAddChain4x16, mulAddChain4x8
	if fused {
		f16, f8 = fmaChain4x16, fmaChain4x8
	}
	base := unsafe.Pointer(unsafe.SliceData(x))
	var buf [chainCols * 16]float32
	for r := 0; r < rows; {
		width := 16
		if rows-r < 16 {
			width = 8
		}
		for c := 0; c < chainCols; c++ {
			copy(buf[c*width:(c+1)*width], acc[c*rows+r:c*rows+r+width])
		}
		xp := (*float32)(unsafe.Add(base, 4*(xbase+r)))
		f := f16
		if width == 8 {
			f = f8
		}
		f(&buf[0], xp, &xoff[0], &w[0][0], &w[1][0], &w[2][0], &w[3][0], uintptr(wstep), uintptr(steps))
		for c := 0; c < chainCols; c++ {
			copy(acc[c*rows+r:c*rows+r+width], buf[c*width:(c+1)*width])
		}
		r += width
	}
}

//go:noescape
func fmaChain1x32(acc, x *float32, xoff *int32, w *float32, wstep, steps uintptr)

//go:noescape
func fmaChain1x8(acc, x *float32, xoff *int32, w *float32, wstep, steps uintptr)

func fmaChainColAsm(acc []float32, x []float32, xbase int, xoff []int32, w []float32, wstep int) {
	rows := len(acc)
	lo, hi := int(xoff[0]), int(xoff[0])
	for _, o := range xoff {
		lo, hi = min(lo, int(o)), max(hi, int(o))
	}
	if xbase+lo < 0 || xbase+hi+rows > len(x) || (len(xoff)-1)*wstep >= len(w) {
		panic("onnx: FMA chain operands out of range")
	}
	base := unsafe.Pointer(unsafe.SliceData(x))
	for r := 0; r < rows; {
		xp := (*float32)(unsafe.Add(base, 4*(xbase+r)))
		if rows-r >= 32 {
			fmaChain1x32(&acc[r], xp, &xoff[0], &w[0], uintptr(wstep), uintptr(len(xoff)))
			r += 32
		} else {
			fmaChain1x8(&acc[r], xp, &xoff[0], &w[0], uintptr(wstep), uintptr(len(xoff)))
			r += 8
		}
	}
}

//go:noescape
func fmaChain4x16W(acc, x *float32, xoff *int32, w0, w1, w2, w3 *float32, woff *int32, steps uintptr)

//go:noescape
func fmaChain4x8W(acc, x *float32, xoff *int32, w0, w1, w2, w3 *float32, woff *int32, steps uintptr)

// fmaChainsIndexedAsm is fmaChainsAsm with per-step weight offsets: column
// c reads w[c][woff[j]] at step j.
func fmaChainsIndexedAsm(acc []float32, rows int, x []float32, xbase int, xoff []int32, w [chainCols][]float32, woff []int32) {
	steps := len(xoff)
	if len(woff) != steps {
		panic("onnx: FMA chain offsets differ")
	}
	lo, hi := int(xoff[0]), int(xoff[0])
	for _, o := range xoff {
		lo, hi = min(lo, int(o)), max(hi, int(o))
	}
	wlo, whi := int(woff[0]), int(woff[0])
	for _, o := range woff {
		wlo, whi = min(wlo, int(o)), max(whi, int(o))
	}
	if xbase+lo < 0 || xbase+hi+rows > len(x) || len(acc) < chainCols*rows || wlo < 0 {
		panic("onnx: FMA chain input out of range")
	}
	for c := range w {
		if whi >= len(w[c]) {
			panic("onnx: FMA chain weight out of range")
		}
	}
	base := unsafe.Pointer(unsafe.SliceData(x))
	var buf [chainCols * 16]float32
	for r := 0; r < rows; {
		width := 16
		if rows-r < 16 {
			width = 8
		}
		for c := 0; c < chainCols; c++ {
			copy(buf[c*width:(c+1)*width], acc[c*rows+r:c*rows+r+width])
		}
		xp := (*float32)(unsafe.Add(base, 4*(xbase+r)))
		f := fmaChain4x16W
		if width == 8 {
			f = fmaChain4x8W
		}
		f(&buf[0], xp, &xoff[0], &w[0][0], &w[1][0], &w[2][0], &w[3][0], &woff[0], uintptr(steps))
		for c := 0; c < chainCols; c++ {
			copy(acc[c*rows+r:c*rows+r+width], buf[c*width:(c+1)*width])
		}
		r += width
	}
}
