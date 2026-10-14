package onnx

import "fmt"

// MKLSmallKNormalN reproduces the original MKL worker at RVA 0x789270,
// for N,N and alpha=beta=1 (the original FeatNet convolution path).
// Storage and alignment follow MKLGEMMNormalN; initial is required.
func MKLSmallKNormalN(a, b, initial []float32, m, n, k, lda, ldb, ldc, cAlignment int) ([]float32, error) {
	if m < 1 || n < 1 || k < 1 || m > int(^uint(0)>>1)/n || lda < m || ldb < k || ldc < m || cAlignment < 0 || cAlignment >= 16 || cAlignment%4 != 0 {
		return nil, fmt.Errorf("unsupported original small-K GEMM geometry or alignment")
	}
	if m > len(a) || k-1 > (len(a)-m)/lda || k > len(b) || n-1 > (len(b)-k)/ldb || m > len(initial) || n-1 > (len(initial)-m)/ldc {
		return nil, fmt.Errorf("original small-K GEMM operands too short")
	}
	if m < 8 || n < 8 || k < 8 {
		return MKLGEMMNormalN(a, b, initial, m, n, k, lda, ldb, ldc, cAlignment, 1, 1)
	}
	y := make([]float32, m*n)
	for col := 0; col < n; col++ {
		copy(y[col*m:], initial[col*ldc:col*ldc+m])
	}
	packedM, packedN := m&^7, n&^3
	firstK := k % 112
	if firstK == 0 {
		firstK = 112
	}
	// The final, possibly short K block runs first, then full blocks run
	// from the beginning. Each packed block adds its dot product to C.
	blocks := [][2]int{{k - firstK, firstK}}
	for start := 0; start < k-firstK; start += 112 {
		blocks = append(blocks, [2]int{start, 112})
	}
	for _, block := range blocks {
		start, width := block[0], block[1]
		for col := 0; col < packedN; col++ {
			for row := 0; row < packedM; row++ {
				sum := float32(0)
				for inner := start; inner < start+width; inner++ {
					sum += float32(a[inner*lda+row] * b[col*ldb+inner])
				}
				y[col*m+row] = sum + y[col*m+row]
			}
		}
		if packedM < m {
			// Keep original ldc and address alignment while updating only
			// the remaining rows. Go's output has contiguous columns.
			seed := make([]float32, (packedN-1)*ldc+m-packedM)
			for col := 0; col < packedN; col++ {
				copy(seed[col*ldc:], y[col*m+packedM:(col+1)*m])
			}
			tail, err := MKLGEMMNormalN(a[start*lda+packedM:], b[start:], seed, m-packedM, packedN, width, lda, ldb, ldc, (cAlignment+packedM*4)%16, 1, 1)
			if err != nil {
				return nil, err
			}
			for col := 0; col < packedN; col++ {
				copy(y[col*m+packedM:(col+1)*m], tail[col*(m-packedM):(col+1)*(m-packedM)])
			}
		}
	}
	for col := packedN; col < n; col++ {
		values, err := MKLGEMVNormal(a, b[col*ldb:col*ldb+k], initial[col*ldc:col*ldc+m], m, k, lda, (cAlignment+(col*ldc%4)*4)%16, 1, 1)
		if err != nil {
			return nil, err
		}
		copy(y[col*m:], values)
	}
	return y, nil
}
