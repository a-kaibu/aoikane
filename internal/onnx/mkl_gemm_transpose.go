package onnx

import "fmt"

// MKLGEMMNormalT reproduces the original generic N,T SGEMM entry.
// B is physically n-by-k column-major; C alignment describes the original.
func MKLGEMMNormalT(a, b, initial []float32, m, n, k, lda, ldb, ldc, cAlignment int, alpha, beta float32) ([]float32, error) {
	if m < 1 || n < 1 || k < 1 || m > int(^uint(0)>>1)/n || lda < m || ldb < n || ldc < m || n > len(b) || k-1 > (len(b)-n)/ldb {
		return nil, fmt.Errorf("unsupported original N,T GEMM geometry")
	}
	packed := make([]float32, n*k)
	for j := 0; j < n; j++ {
		for i := 0; i < k; i++ {
			packed[j*k+i] = b[i*ldb+j]
		}
	}
	return MKLGEMMNormalN(a, packed, initial, m, n, k, lda, k, ldc, cAlignment, alpha, beta)
}

// MKLSmallKNormalT uses the original small-K worker's packed blocks and
// generic row/column tails. beta=0 or 1, alpha=1, as in native ConvTranspose.
func MKLSmallKNormalT(a, b, initial []float32, m, n, k, lda, ldb, ldc, cAlignment int, beta float32) ([]float32, error) {
	if m < 1 || n < 1 || k < 1 || m > int(^uint(0)>>1)/n || lda < m || ldb < n || ldc < m || n-1 > (int(^uint(0)>>1)-m)/ldc || n > len(b) || k-1 > (len(b)-n)/ldb || (beta != 0 && beta != 1) {
		return nil, fmt.Errorf("unsupported original small-K N,T GEMM geometry")
	}
	packed := make([]float32, n*k)
	for j := 0; j < n; j++ {
		for i := 0; i < k; i++ {
			packed[j*k+i] = b[i*ldb+j]
		}
	}
	if beta == 0 {
		initial = make([]float32, (n-1)*ldc+m)
	}
	y, err := MKLSmallKNormalN(a, packed, initial, m, n, k, lda, k, ldc, cAlignment)
	if err != nil {
		return nil, err
	}
	if m >= 8 && n >= 8 && k >= 8 {
		// Original column tails use GEMV with incx=ldb. That strided-X
		// path keeps a sequential accumulator even in the unaligned output
		// prefix, unlike the contiguous-X path's two groups of four.
		for j := n &^ 3; j < n; j++ {
			for row := 0; row < m; row++ {
				acc := initial[j*ldc+row]
				for i := 0; i < k; i++ {
					acc += float32(a[i*lda+row] * b[i*ldb+j])
				}
				y[j*m+row] = acc
			}
		}
	}
	return y, nil
}
