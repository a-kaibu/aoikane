package onnx

import "fmt"

// MKLGEMVNormal reproduces the original MKL 2020.0.2 non-transposed float32
// GEMV at RVA 0x86f060, with contiguous x and y. A has m rows, n columns,
// and lda spacing. yAlignment is the ORIGINAL output address modulo 16.
// initial may be nil only when beta=0. Result values are contiguous.
func MKLGEMVNormal(a, x, initial []float32, m, n, lda, yAlignment int, alpha, beta float32) ([]float32, error) {
	if m < 1 || n < 1 || lda < m || yAlignment < 0 || yAlignment >= 16 || yAlignment%4 != 0 {
		return nil, fmt.Errorf("unsupported original normal GEMV geometry or alignment")
	}
	if m > len(a) || n-1 > (len(a)-m)/lda || len(x) < n || (beta != 0 && len(initial) != m) {
		return nil, fmt.Errorf("original normal GEMV operands too short")
	}
	y := make([]float32, m)
	if beta != 0 {
		copy(y, initial)
		if beta != 1 {
			for i := range y {
				y[i] = float32(beta * y[i])
			}
		}
	}
	if alpha == 0 {
		return y, nil
	}
	prefix := (-yAlignment / 4) & 3
	if m < prefix+4 {
		prefix = 0 // No SIMD body: all output rows use the sequential tail.
	}
	col := 0
	for ; col+8 <= n; col += 8 {
		var scaled [8]float32
		for lane := range scaled {
			scaled[lane] = float32(alpha * x[col+lane])
		}
		for row := range y {
			first := float32(a[col*lda+row]*scaled[0]) + y[row]
			if row < prefix {
				for lane := 1; lane < 4; lane++ {
					first += float32(a[(col+lane)*lda+row] * scaled[lane])
				}
				second := float32(a[(col+4)*lda+row] * scaled[4])
				for lane := 5; lane < 8; lane++ {
					second += float32(a[(col+lane)*lda+row] * scaled[lane])
				}
				y[row] = second + first
			} else {
				for lane := 1; lane < 8; lane++ {
					first += float32(a[(col+lane)*lda+row] * scaled[lane])
				}
				y[row] = first
			}
		}
	}
	for ; col < n; col++ {
		scaled := float32(alpha * x[col])
		for row := range y {
			y[row] = float32(a[col*lda+row]*scaled) + y[row]
		}
	}
	return y, nil
}
