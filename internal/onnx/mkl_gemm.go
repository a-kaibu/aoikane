package onnx

import "fmt"

// MKLGEMMTransposeT reproduces the original generic SGEMM T,T path, beta=0.
// A stores m columns of k elements with lda spacing; B stores k columns of
// n elements with ldb spacing. Output is column-major with m rows. The native
// four-output SIMD lanes are independent sequential dot products.
func MKLGEMMTransposeT(a, b []float32, m, n, k, lda, ldb int, alpha float32) ([]float32, error) {
	if m < 1 || n < 1 || k < 1 || lda < k || ldb < n {
		return nil, fmt.Errorf("unsupported original T,T GEMM geometry")
	}
	if k > len(a) || m-1 > (len(a)-k)/lda || n > len(b) || k-1 > (len(b)-n)/ldb {
		return nil, fmt.Errorf("original T,T GEMM operands too short")
	}
	y := make([]float32, m*n)
	for col := 0; col < n; col++ {
		for row := 0; row < m; row++ {
			sum := float32(0)
			for i := 0; i < k; i++ {
				sum += float32(a[row*lda+i] * b[col+i*ldb])
			}
			y[col*m+row] = float32(alpha * sum)
		}
	}
	return y, nil
}

// MKLGEMMTransposeN reproduces the original MKL 2020.0.2 generic SGEMM
// T,N kernel at torch_cpu.dll RVA 0x2824320, with beta=0. A stores m
// contiguous columns of k elements (lda spacing), and B stores n columns
// of k elements (ldb spacing). The result is column-major with m rows.
// bAlignment is the original B address modulo 16, independent of Go memory.
// This kernel is separate from the ONNX/MLAS backend.
func MKLGEMMTransposeN(a, b []float32, m, n, k, lda, ldb, bAlignment int, alpha float32) ([]float32, error) {
	if m < 1 || n < 1 || k < 1 || lda < k || ldb < k || bAlignment < 0 || bAlignment >= 16 || bAlignment%4 != 0 {
		return nil, fmt.Errorf("unsupported original GEMM geometry or alignment")
	}
	if k > len(a) || m-1 > (len(a)-k)/lda || k > len(b) || n-1 > (len(b)-k)/ldb {
		return nil, fmt.Errorf("original GEMM operands too short")
	}
	y := make([]float32, m*n)
	for col := 0; col < n; col++ {
		prefix := (-(bAlignment/4 + col*ldb)) & 3
		for row := 0; row < m; row++ {
			product := func(i int) float32 { return float32(a[row*lda+i] * b[col*ldb+i]) }
			sum := float32(0)
			i := 0
			block := 4
			if row >= m&^3 {
				block = 8
			}
			if k >= prefix+block {
				for ; i < prefix; i++ {
					sum += product(i)
				}
				low, high := [4]float32{sum}, [4]float32{}
				for ; i+block <= k; i += block {
					for lane := range low {
						low[lane] += product(i + lane)
						if block == 8 {
							high[lane] += product(i + 4 + lane)
						}
					}
				}
				if block == 8 {
					for lane := range low {
						low[lane] += high[lane]
					}
				}
				sum = float32(low[0]+low[2]) + float32(low[1]+low[3])
			}
			for ; i < k; i++ {
				sum += product(i)
			}
			y[col*m+row] = float32(alpha * sum)
		}
	}
	return y, nil
}
