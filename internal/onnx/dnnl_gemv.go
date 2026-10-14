package onnx

import "fmt"

// Adapted from oneDNN v2.7.3 gemv_driver.cpp and
// jit_avx_gemv_t_f32_kern.cpp (Copyright 2019-2020 Intel Corporation).
// Licensed under Apache-2.0; see ONEDNN-LICENSE.
//
// dnnlGEMVRow reproduces jump_to_gemv for N,N GEMM with m=1.
// The transposed AVX2 GEMV reduces eight FMA lanes after each 16-wide
// main loop, then adds the 8/4/2/1 remainders separately. This dispatch
// precedes the no-copy GEMM driver, whose m=1 order is different.
func dnnlGEMVRow(a, b, initial []float32, n, k, lda, ldb, ldc int, beta float32) ([]float32, error) {
	if n < 1 || k < 1 || lda < 1 || ldb < k || ldc < 1 || (beta != 0 && beta != 1) {
		return nil, fmt.Errorf("unsupported original oneDNN row GEMV geometry/beta")
	}
	if len(a) < 1 || k-1 > (len(a)-1)/lda || len(b) < k || n-1 > (len(b)-k)/ldb || (beta == 1 && (len(initial) < 1 || n-1 > (len(initial)-1)/ldc)) {
		return nil, fmt.Errorf("original oneDNN row GEMV operands too short")
	}
	reduce := func(v [8]float32) float32 {
		lo := float32(float32(v[0]+v[1]) + float32(v[4]+v[5]))
		hi := float32(float32(v[2]+v[3]) + float32(v[6]+v[7]))
		return float32(lo + hi)
	}
	y := make([]float32, n)
	for col := range y {
		if beta == 1 {
			y[col] = initial[col*ldc]
		}
		var lanes [8]float32
		main := k &^ 15
		for i := 0; i < main; i++ {
			lanes[i%8] = fma32(a[i*lda], b[col*ldb+i], lanes[i%8])
		}
		if main != 0 {
			y[col] = fma32(1, reduce(lanes), y[col])
		}
		offset := main
		for width := 8; width > 0; width >>= 1 {
			if k&width == 0 {
				continue
			}
			lanes = [8]float32{}
			for lane := 0; lane < width; lane++ {
				lanes[lane] = fma32(a[(offset+lane)*lda], b[col*ldb+offset+lane], 0)
			}
			y[col] = fma32(1, reduce(lanes), y[col])
			offset += width
		}
	}
	return y, nil
}
