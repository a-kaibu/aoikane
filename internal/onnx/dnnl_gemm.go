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
	// Every element is assigned by the first K block (beta 0) or copied
	// from initial (beta 1) before it is accumulated.
	y := getFloatsUncleared(m * n)
	if beta == 1 {
		for col := 0; col < n; col++ {
			copy(y[col*m:], initial[col*ldc:col*ldc+m])
		}
	}
	blockK, bRowStride, bColStride := 256, 1, ldb
	if transposeB {
		blockK, bRowStride, bColStride = 96, ldb, 1
	}
	var tile [6 * 16]float32
	var wide []float32
	var tail [6 * 8]float32
	for bk := 0; bk < k; {
		ks := k - bk
		if ks >= 2*blockK {
			ks = blockK
		} else if ks > blockK {
			ks = (ks + 1) / 2
		}
		var all, even, odd []int32
		if useChainedKernels {
			for inner := 0; inner < ks; inner++ {
				all = append(all, int32((bk+inner)*lda))
				if inner < ks&^1 {
					if inner%2 == 0 {
						even = append(even, int32((bk+inner)*lda))
					} else {
						odd = append(odd, int32((bk+inner)*lda))
					}
				}
			}
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
				// Full 16-row tiles share their accumulation flags per
				// column tile, so their lanes run in one vector pass.
				firstRow := 0
				if useChainedKernels && ms >= 16 {
					firstRow = ms &^ 15
					if cap(wide) < 6*firstRow {
						wide = make([]float32, 6*firstRow)
					}
					for col := 0; col < ns; col += 6 {
						nr := min(6, ns-col)
						dual, fused := nr <= 3, nr >= 3
						v := wide[:nr*firstRow]
						dnnlGEMMTileChains(v, a, b, bm, firstRow, bn+col, nr, bk, ks, bColStride, bRowStride, dual, fused, all, even, odd)
						for cc := 0; cc < nr; cc++ {
							dst := y[(bn+col+cc)*m+bm : (bn+col+cc)*m+bm+firstRow]
							src := v[cc*firstRow : (cc+1)*firstRow]
							if bk == 0 && beta == 0 {
								copy(dst, src)
							} else {
								for i := range dst {
									dst[i] += src[i]
								}
							}
						}
					}
				}
				for row := firstRow; row < ms; row += 16 {
					mr := min(16, ms-row)
					for col := 0; col < ns; col += 6 {
						nr := min(6, ns-col)
						dual := mr <= 8 || nr <= 3
						fused := nr >= 3
						// Vector lanes are rows; each lane keeps the scalar
						// accumulator order below.
						vec := 0
						if useChainedKernels {
							vec = mr &^ 7
						}
						if vec > 0 {
							dnnlGEMMTileChains(tile[:], a, b, bm+row, vec, bn+col, nr, bk, ks, bColStride, bRowStride, dual, fused, all, even, odd)
						}
						// Remaining rows reuse an eight-lane pass ending at the
						// tile's last row; lanes are independent, so the earlier
						// rows it also computes are simply not used.
						overlap := useChainedKernels && mr%8 != 0 && row+mr >= 8
						if overlap {
							dnnlGEMMTileChains(tail[:], a, b, bm+row+mr-8, 8, bn+col, nr, bk, ks, bColStride, bRowStride, dual, fused, all, even, odd)
						}
						for cc := 0; cc < nr; cc++ {
							for rr := 0; rr < mr; rr++ {
								var v0 float32
								if rr < vec {
									v0 = tile[cc*vec+rr]
								} else if overlap {
									v0 = tail[cc*8+rr-(mr-8)]
								} else {
									v1 := float32(0)
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

// dnnlGEMMTileChains computes the accumulators of rows [row, row+rows) and
// columns [col, col+cols) of one micro tile into tile[cc*rows+rr], in the
// same order as the scalar loop of dnnlGEMM. rows is a multiple of 8.
func dnnlGEMMTileChains(tile, a, b []float32, row, rows, col, cols, bk, ks, bColStride, bRowStride int, dual, fused bool, all, even, odd []int32) {
	buf := [2][]float32{make([]float32, chainCols*rows), make([]float32, chainCols*rows)}
	for group := 0; group < cols; group += chainCols {
		var w, w1 [chainCols][]float32
		for cc := range w {
			index := (col+group+min(cc, cols-group-1))*bColStride + bk*bRowStride
			w[cc] = b[index:]
			w1[cc] = b[min(index+bRowStride, len(b)-1):]
		}
		v0, v1 := buf[0][:chainCols*rows], buf[1][:chainCols*rows]
		clear(v0)
		if !dual {
			fmaChains(v0, rows, a, row, all, w, bRowStride, fused)
		} else {
			clear(v1)
			if len(even) > 0 {
				fmaChains(v0, rows, a, row, even, w, 2*bRowStride, fused)
			}
			if len(odd) > 0 {
				fmaChains(v1, rows, a, row, odd, w1, 2*bRowStride, fused)
			}
			for i := range v0 {
				v0[i] += v1[i]
			}
			if ks%2 != 0 {
				tail := all[ks-1:]
				for cc := range w {
					w[cc] = w[cc][(ks-1)*bRowStride:]
				}
				fmaChains(v0, rows, a, row, tail, w, bRowStride, fused)
			}
		}
		for cc := 0; cc < min(chainCols, cols-group); cc++ {
			copy(tile[(group+cc)*rows:(group+cc+1)*rows], v0[cc*rows:(cc+1)*rows])
		}
	}
}
