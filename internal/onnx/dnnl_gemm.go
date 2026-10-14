package onnx

import "fmt"

// DNNLGEMMNormalN reproduces original oneDNN's AVX2 N,N no-copy GEMM
// driver at RVA 0xb36f4c0, with alpha=1 and beta=0 or 1. A/B/initial use
// column-major storage; result columns are contiguous without C padding.
func DNNLGEMMNormalN(a, b, initial []float32, m, n, k, lda, ldb, ldc int, beta float32) ([]float32, error) {
	return dnnlGEMM(a, b, initial, m, n, k, lda, ldb, ldc, beta, false)
}

func dnnlGEMM(a, b, initial []float32, m, n, k, lda, ldb, ldc int, beta float32, transposeB bool) ([]float32, error) {
	bRows, bCols := k, n
	if transposeB {
		bRows, bCols = n, k
	}
	if m < 1 || n < 1 || k < 1 || m > int(^uint(0)>>1)/n || lda < m || ldb < bRows || ldc < m || (beta != 0 && beta != 1) {
		return nil, fmt.Errorf("unsupported original oneDNN GEMM geometry/beta")
	}
	if m > len(a) || k-1 > (len(a)-m)/lda || bRows > len(b) || bCols-1 > (len(b)-bRows)/ldb || (beta != 0 && (m > len(initial) || n-1 > (len(initial)-m)/ldc)) {
		return nil, fmt.Errorf("original oneDNN GEMM operands too short")
	}
	y := make([]float32, m*n)
	if beta == 1 {
		for col := 0; col < n; col++ {
			copy(y[col*m:], initial[col*ldc:col*ldc+m])
		}
	}
	blockK, bRowStride, bColStride := 256, 1, ldb
	if transposeB {
		blockK, bRowStride, bColStride = 96, ldb, 1
	}
	for bk := 0; bk < k; {
		ks := k - bk
		if ks >= 2*blockK {
			ks = blockK
		} else if ks > blockK {
			ks = (ks + 1) / 2
		}
		for bm := 0; bm < m; {
			ms := m - bm
			if ms >= 8064 {
				ms = 4032
			} else if ms > 6048 {
				ms = (ms + 1) / 2
			}
			for bn := 0; bn < n; {
				ns := n - bn
				if ns >= 96 {
					ns = 48
				} else if ns > 72 {
					ns = (ns + 1) / 2
				}
				for row := 0; row < ms; row += 16 {
					mr := min(16, ms-row)
					for col := 0; col < ns; col += 6 {
						nr := min(6, ns-col)
						dual := mr <= 8 || nr <= 3
						fused := nr >= 3
						for cc := 0; cc < nr; cc++ {
							for rr := 0; rr < mr; rr++ {
								v0, v1 := float32(0), float32(0)
								inner := 0
								for ; inner < ks&^1; inner++ {
									x, w := a[(bk+inner)*lda+bm+row+rr], b[(bn+col+cc)*bColStride+(bk+inner)*bRowStride]
									acc := &v0
									if dual && inner%2 != 0 {
										acc = &v1
									}
									if fused {
										*acc = fma32(x, w, *acc)
									} else {
										*acc += float32(x * w)
									}
								}
								if dual {
									v0 += v1
								}
								if inner < ks {
									x, w := a[(bk+inner)*lda+bm+row+rr], b[(bn+col+cc)*bColStride+(bk+inner)*bRowStride]
									if fused {
										v0 = fma32(x, w, v0)
									} else {
										v0 += float32(x * w)
									}
								}
								index := (bn+col+cc)*m + bm + row + rr
								if bk == 0 && beta == 0 {
									y[index] = v0
								} else {
									y[index] += v0
								}
							}
						}
					}
				}
				bn += ns
			}
			bm += ms
		}
		bk += ks
	}
	return y, nil
}
