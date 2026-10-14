package onnx

import "fmt"

// MKLGEMMNormalN reproduces the original MKL generic N,N SGEMM kernel.
// A, B and initial use column-major storage, including leading dimensions.
// cAlignment is the ORIGINAL C address modulo 16; ldc also affects alignment.
// Result columns are contiguous (m*n values), without the original C padding.
func MKLGEMMNormalN(a, b, initial []float32, m, n, k, lda, ldb, ldc, cAlignment int, alpha, beta float32) ([]float32, error) {
	if m < 1 || n < 1 || k < 1 || lda < m || ldb < k || ldc < m || cAlignment < 0 || cAlignment >= 16 || cAlignment%4 != 0 {
		return nil, fmt.Errorf("unsupported original normal GEMM geometry or alignment")
	}
	if m > len(a) || k-1 > (len(a)-m)/lda || k > len(b) || n-1 > (len(b)-k)/ldb || (beta != 0 && (m > len(initial) || n-1 > (len(initial)-m)/ldc)) {
		return nil, fmt.Errorf("original normal GEMM operands too short")
	}
	y := make([]float32, m*n)
	for col := 0; col < n; col++ {
		var seed []float32
		if beta != 0 {
			seed = initial[col*ldc : col*ldc+m]
		}
		values, err := MKLGEMVNormal(a, b[col*ldb:col*ldb+k], seed, m, k, lda, (cAlignment+(col*ldc%4)*4)%16, alpha, beta)
		if err != nil {
			return nil, err
		}
		copy(y[col*m:], values)
	}
	return y, nil
}
