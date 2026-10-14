package onnx

import "fmt"

// MKLGEMVTranspose reproduces the observed transposed float32 GEMV kernel in
// the original torch_cpu.dll (MKL 2020.0.2, RVA 0x86f060). Matrix columns have
// m elements and lda spacing; x has incx spacing. Alignment is the ORIGINAL
// operand address modulo 16, not the address of the Go slices. The supported
// operation has beta=0. This is kept separate from the ONNX/MLAS backend.
func MKLGEMVTranspose(a, x []float32, m, n, lda, incx, aAlignment, xAlignment int, alpha float32) ([]float32, error) {
	if m < 1 || n < 1 || lda < m || incx < 1 || aAlignment < 0 || aAlignment >= 16 || aAlignment%4 != 0 || xAlignment < 0 || xAlignment >= 16 || xAlignment%4 != 0 {
		return nil, fmt.Errorf("unsupported original GEMV geometry or alignment")
	}
	if m > len(a) || n-1 > (len(a)-m)/lda || len(x) < 1 || m-1 > (len(x)-1)/incx {
		return nil, fmt.Errorf("original GEMV operands too short")
	}
	y := make([]float32, n)
	if alpha == 0 {
		return y, nil
	}
	product := func(col, k int) float32 { return float32(a[col*lda+k] * x[k*incx]) }
	horizontal := func(v [4]float32) float32 { return float32(v[0]+v[2]) + float32(v[1]+v[3]) }
	finish := func(col int, sum float32) { y[col] = float32(alpha*sum) + float32(0) }
	col := 0
	for ; col+4 <= n; col += 4 {
		prefix := (-(aAlignment/4 + (col+3)*lda + 1)) & 3
		for j := 0; j < 4; j++ {
			sum := product(col+j, 0)
			k := 1
			if m-1 >= prefix+4 {
				for ; k < 1+prefix; k++ {
					sum += product(col+j, k)
				}
				v := [4]float32{sum}
				for ; k+4 <= m; k += 4 {
					for lane := range v {
						v[lane] += product(col+j, k+lane)
					}
				}
				sum = horizontal(v)
			}
			for ; k < m; k++ {
				sum += product(col+j, k)
			}
			finish(col+j, sum)
		}
	}
	if incx != 1 {
		for ; col < n; col++ {
			v := [8]float32{product(col, 0)}
			k := 1
			for ; k+8 <= m; k += 8 {
				for lane := range v {
					v[lane] += product(col, k+lane)
				}
			}
			sum := v[0]
			for _, lane := range v[1:] {
				sum += lane
			}
			// The native fallthrough tail visits the remaining inputs backwards.
			for tail := m - 1; tail >= k; tail-- {
				sum += product(col, tail)
			}
			finish(col, sum)
		}
		return y, nil
	}
	// Remaining contiguous outputs run in pairs. The first output uses two
	// four-lane accumulators; the second uses one, updated twice per block.
	for ; col < n; col += 2 {
		width := min(2, n-col)
		prefix := (-(xAlignment/4 + 1)) & 3
		for j := 0; j < width; j++ {
			sum := product(col+j, 0)
			k := 1
			if m-1 >= prefix+8 {
				for ; k < 1+prefix; k++ {
					sum += product(col+j, k)
				}
				low, high := [4]float32{sum}, [4]float32{}
				for ; k+8 <= m; k += 8 {
					for lane := range low {
						low[lane] += product(col+j, k+lane)
						if j == 0 {
							high[lane] += product(col+j, k+4+lane)
						} else {
							low[lane] += product(col+j, k+4+lane)
						}
					}
				}
				if j == 0 {
					for lane := range low {
						low[lane] += high[lane]
					}
				}
				sum = horizontal(low)
			}
			for ; k < m; k++ {
				sum += product(col+j, k)
			}
			finish(col+j, sum)
		}
	}
	return y, nil
}
