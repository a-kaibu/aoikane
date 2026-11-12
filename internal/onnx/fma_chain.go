package onnx

// FMA chain kernels evaluate many independent sequential reductions at once.
// Each output lane r of column c computes, for j in order,
//
//	acc[c][r] = fma(x[xoff[j]+r], w[c][j*wstep], acc[c][r])
//
// (or acc + round(x*w) for the unfused variant). Every lane keeps the exact
// scalar order of the reconstructed kernels; SIMD only evaluates independent
// lanes side by side, and hardware FMA rounds once like fma32.

// chainCols is the number of weight columns evaluated per kernel call.
const chainCols = 4

// fmaChains runs rows (a multiple of 8) lanes for chainCols columns. acc is
// laid out acc[c*rows+r]; unused columns may repeat a weight pointer.
func fmaChains(acc []float32, rows int, x []float32, xbase int, xoff []int32, w [chainCols][]float32, wstep int, fused bool) {
	if rows <= 0 || len(xoff) == 0 {
		return
	}
	if useAVX2FMA {
		fmaChainsAsm(acc, rows, x, xbase, xoff, w, wstep, fused)
		return
	}
	if fastMath.Load() && fmaChainsSIMD(acc, rows, x, xbase, xoff, w, wstep, fused) {
		return
	}
	for c := 0; c < chainCols; c++ {
		wc := w[c]
		for r := 0; r < rows; r++ {
			a := acc[c*rows+r]
			for j, o := range xoff {
				v := x[xbase+int(o)+r]
				if fused {
					a = fma32(v, wc[j*wstep], a)
				} else {
					a = float32(a + float32(v*wc[j*wstep]))
				}
			}
			acc[c*rows+r] = a
		}
	}
}

// fmaChainCol is the single-column form: acc[r] = fma(x[xbase+xoff[j]+r],
// w[j*wstep], acc[r]) for j in order. len(acc) must be a multiple of 8.
func fmaChainCol(acc []float32, x []float32, xbase int, xoff []int32, w []float32, wstep int) {
	if len(acc) == 0 || len(xoff) == 0 {
		return
	}
	if useAVX2FMA {
		fmaChainColAsm(acc, x, xbase, xoff, w, wstep)
		return
	}
	if fastMath.Load() && fmaChainColSIMD(acc, x, xbase, xoff, w, wstep) {
		return
	}
	for r := range acc {
		a := acc[r]
		for j, o := range xoff {
			a = fma32(x[xbase+int(o)+r], w[j*wstep], a)
		}
		acc[r] = a
	}
}

// fmaChainsIndexed is fmaChains (fused) with weight w[c][woff[j]] at step j.
func fmaChainsIndexed(acc []float32, rows int, x []float32, xbase int, xoff []int32, w [chainCols][]float32, woff []int32) {
	if rows <= 0 || len(xoff) == 0 {
		return
	}
	if useAVX2FMA {
		fmaChainsIndexedAsm(acc, rows, x, xbase, xoff, w, woff)
		return
	}
	if fastMath.Load() && fmaChainsIndexedSIMD(acc, rows, x, xbase, xoff, w, woff) {
		return
	}
	for c := 0; c < chainCols; c++ {
		wc := w[c]
		for r := 0; r < rows; r++ {
			a := acc[c*rows+r]
			for j, o := range xoff {
				a = fma32(x[xbase+int(o)+r], wc[woff[j]], a)
			}
			acc[c*rows+r] = a
		}
	}
}
