// Copyright 2018-2020 Intel Corporation
// Copyright 2026 aoikane contributors
// SPDX-License-Identifier: Apache-2.0
// Partitioning adapted from oneDNN v2.7 gemm_utils_f32.cpp; see ONEDNN-LICENSE.
package onnx

import "fmt"

// originalGEMMNTPartition reproduces calc_nthr_nocopy_avx for the original
// six-thread runtime, without K splitting (all vocoder inputs have K<=256).
// Adapted from oneDNN v2.7 gemm_utils_f32.cpp, Apache-2.0; see ONEDNN-LICENSE.
func originalGEMMNTPartition(m, n int, profile ...int) (int, int) {
	if originalThreadCount(profile) == 1 {
		return (m + 15) &^ 15, n
	}
	mt, nt := (m+63)/64, (n+47)/48
	if mt == 1 {
		nt = 6
	}
	if nt == 1 {
		mt = 6
	}
	for mt*nt > 6 {
		if mt > nt {
			mt--
		} else {
			nt--
		}
	}
	for mt*nt < 6 {
		if mt < nt {
			mt++
		} else {
			nt++
		}
	}
	if mt*nt > 6 && mt > 1 && nt > 1 {
		if mt <= nt {
			mt = min(2, (m+15)/16)
			nt = 6 / mt
			for mt > 1 && mt*nt != 6 {
				mt--
				nt = 6 / mt
			}
		} else {
			nt = min(2, n)
			mt = 6 / nt
			for nt > 1 && mt*nt != 6 {
				nt--
				mt = 6 / nt
			}
		}
	}
	return ((m+mt-1)/mt + 15) &^ 15, (n + nt - 1) / nt
}

func graphConvTranspose(n graphNode, in []Value, original bool, profile ...int) ([]Value, error) {
	threads := originalThreadCount(profile)
	if !original {
		return nil, fmt.Errorf("ConvTranspose requires original kernels")
	}
	if len(in) != 3 {
		return nil, fmt.Errorf("ConvTranspose requires input, weight and bias")
	}
	x, w, b := in[0], in[1], in[2]
	if x.Kind != 1 || w.Kind != 1 || b.Kind != 1 || len(x.Shape) != 3 || len(w.Shape) != 3 || len(b.Shape) != 1 || x.Shape[0] != 1 || x.Shape[1] != w.Shape[0] || b.Shape[0] != w.Shape[1] {
		return nil, fmt.Errorf("unsupported ConvTranspose tensor shapes")
	}
	ci, co, kw, frames := int(w.Shape[0]), int(w.Shape[1]), int(w.Shape[2]), int(x.Shape[2])
	if ci < 1 || ci > 256 || co < 1 || co > 128 || kw < 1 || kw > 10 || frames < 1 || frames > 32768 || len(x.F) != ci*frames || len(w.F) != ci*co*kw || len(b.F) != co {
		return nil, fmt.Errorf("unsupported ConvTranspose dimensions")
	}
	c := &Conv1D{Kernel: kw, Stride: 1, Dilation: 1}
	attrs := make(map[string]graphAttr, len(n.attrs))
	opad := 0
	for key, a := range n.attrs {
		if key == "auto_pad" && a.text != "NOTSET" {
			return nil, fmt.Errorf("ConvTranspose auto_pad is not yet implemented")
		}
		if key == "output_padding" {
			if len(a.ints) != 1 || a.ints[0] < 0 || a.ints[0] > 32 {
				return nil, fmt.Errorf("invalid ConvTranspose output_padding")
			}
			opad = int(a.ints[0])
		} else {
			attrs[key] = a
		}
	}
	if err := configureConv(c, attrs); err != nil {
		return nil, err
	}
	// Accept only the five audited model geometries.
	matched := ci == 256 && co == 128 && kw == 10 && c.Stride == 5 && c.Left == 3 && c.Right == 3 && opad == 1 ||
		ci == 128 && co == 64 && kw == 8 && c.Stride == 4 && c.Left == 2 && c.Right == 2 && opad == 0 ||
		ci == 64 && co == 32 && kw == 6 && c.Stride == 3 && c.Left == 2 && c.Right == 2 && opad == 1 ||
		(ci == 32 && co == 16 || ci == 16 && co == 8) && kw == 4 && c.Stride == 2 && c.Left == 1 && c.Right == 1 && opad == 0
	if !matched || c.Dilation != 1 {
		return nil, fmt.Errorf("original ConvTranspose kernel geometry is not yet implemented")
	}
	length := (frames-1)*c.Stride - c.Left - c.Right + kw + opad
	if length < 1 {
		return nil, fmt.Errorf("invalid ConvTranspose output length")
	}
	columns := co * kw
	avx512 := originalAVX512Convolutions(profile)
	if avx512 && len(x.F) > 20480 && ci == 128 {
		return []Value{{Kind: 1, Shape: []int64{1, int64(co), int64(length)}, F: originalAVX512TransposeDirect(x.F, w.F, b.F, ci, co, kw, frames, length, c.Stride, c.Left)}}, nil
	}
	col := make([]float32, frames*columns)
	// For the observed N,T geometries (K=16..256), the AVX2 dispatch
	// threshold decides between no-copy and packed GEMM. Packed K=128 uses
	// one sequential FMA accumulator per output, without the 96-element
	// reductions of the no-copy worker. The audited AVX512 profile uses this
	// order for its remaining large geometries after the direct case above.
	if len(x.F) <= 20480 {
		for _, part := range originalMKLPartitions(max(frames, columns), threads) {
			row, rows, start, count := 0, frames, part[0], part[1]
			if frames > columns {
				row, rows, start, count = part[0], part[1], 0, columns
			}
			v, err := MKLSmallKNormalT(x.F[row:], w.F[start:], nil, rows, count, ci, frames, columns, frames, ((start*frames+row)%4)*4, 0)
			if err != nil {
				return nil, err
			}
			for j := 0; j < count; j++ {
				copy(col[(start+j)*frames+row:], v[j*rows:(j+1)*rows])
			}
		}
	} else if avx512 || 1/float64(frames)+1/float64(columns) < 0.0038 {
		if !avx512 && ci != 128 {
			return nil, fmt.Errorf("original packed ConvTranspose K is not yet implemented")
		}
		for j := 0; j < columns; j++ {
			for t := 0; t < frames; t++ {
				acc := float32(0)
				for i := 0; i < ci; i++ {
					acc = fma32(x.F[i*frames+t], w.F[i*columns+j], acc)
				}
				col[j*frames+t] = acc
			}
		}
	} else {
		mb, nb := originalGEMMNTPartition(frames, columns, threads)
		for fromN := 0; fromN < columns; fromN += nb {
			for fromM := 0; fromM < frames; fromM += mb {
				m, countN := min(mb, frames-fromM), min(nb, columns-fromN)
				v, err := DNNLGEMMNormalT(x.F[fromM:], w.F[fromN:], nil, m, countN, ci, frames, columns, frames, 0)
				if err != nil {
					return nil, err
				}
				for j := 0; j < countN; j++ {
					copy(col[(fromN+j)*frames+fromM:], v[j*m:(j+1)*m])
				}
			}
		}
	}
	y := make([]float32, co*length)
	// oneDNN col2im accumulates kernel offsets before spatial positions.
	for ch := 0; ch < co; ch++ {
		for k := 0; k < kw; k++ {
			for t := 0; t < frames; t++ {
				pos := t*c.Stride - c.Left + k
				if pos >= 0 && pos < length {
					y[ch*length+pos] += col[(ch*kw+k)*frames+t]
				}
			}
		}
		for t := 0; t < length; t++ {
			y[ch*length+t] += b.F[ch]
		}
	}
	return []Value{{Kind: 1, Shape: []int64{1, int64(co), int64(length)}, F: y}}, nil
}
