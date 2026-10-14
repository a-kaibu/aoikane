package onnx

import "fmt"

// RunOriginalLinear executes the original six-thread MKL paths for all thirteen
// FeatNet projections. Inputs/weights are logical row-major, without bias.
// The original projection inputs have physical strides [rows*In,1,rows].
func (l *Linear) RunOriginalLinear(input []float32, rows int) ([]float32, error) {
	return l.runOriginalLinear(input, rows, 6)
}

func (l *Linear) runOriginalLinear(input []float32, rows, threads int) ([]float32, error) {
	if l.Out == 256 {
		return l.runOriginalEmbedding(input, rows, threads)
	}
	if rows < 2 || !((l.In == 64 && (l.Out == 1 || l.Out == 2)) || (l.In == 256 && (l.Out == 5 || l.Out == 60))) || len(input)/l.In != rows || len(input)%l.In != 0 || len(l.weight) != l.In*l.Out || rows > int(^uint(0)>>1)/l.Out {
		return nil, fmt.Errorf("unsupported original projection shape")
	}
	a := make([]float32, len(input))
	w := make([]float32, len(l.weight))
	for row := 0; row < rows; row++ {
		for k := 0; k < l.In; k++ {
			a[k*rows+row] = input[row*l.In+k]
		}
	}
	for col := 0; col < l.Out; col++ {
		for k := 0; k < l.In; k++ {
			w[col*l.In+k] = l.weight[k*l.Out+col]
		}
	}
	if l.Out == 1 {
		return MKLGEMVNormal(a, w, nil, rows, l.In, rows, 0, 1, 0)
	}
	y := make([]float32, rows*l.Out)
	for _, part := range originalMKLPartitions(max(rows, l.Out), threads) {
		col, columns, row, count := part[0], part[1], 0, rows
		if rows >= l.Out {
			col, columns, row, count = 0, l.Out, part[0], part[1]
		}
		if min(columns, count, l.In) < 8 {
			values, err := MKLGEMMTransposeT(w[col*l.In:], a[row:], columns, count, l.In, l.In, rows, 1)
			if err != nil {
				return nil, err
			}
			for r := 0; r < count; r++ {
				copy(y[(row+r)*l.Out+col:], values[r*columns:(r+1)*columns])
			}
			continue
		}
		packedRows := count &^ 3
		// Original packed worker reduces K in 112-element blocks, with
		// the short final block first. Each partial starts from zero.
		blocks := [][2]int{}
		if tail := l.In % 112; tail != 0 {
			blocks = append(blocks, [2]int{l.In - tail, tail})
		}
		for k := 0; k < l.In/112*112; k += 112 {
			blocks = append(blocks, [2]int{k, 112})
		}
		for _, block := range blocks {
			for r := 0; r < packedRows; r++ {
				for c := 0; c < columns; c++ {
					// The unpaired output columns use generic T,T GEMM,
					// whose sequential reduction is the same here.
					sum := float32(0)
					for k := block[0]; k < block[0]+block[1]; k++ {
						sum += float32(a[k*rows+row+r] * w[(col+c)*l.In+k])
					}
					index := (row+r)*l.Out + col + c
					y[index] = float32(sum + y[index])
				}
			}
		}
		for r := packedRows; r < count; r++ {
			values, err := MKLGEMVTranspose(w[col*l.In:], a[row+r:], l.In, columns, l.In, rows, (col*l.In*4)%16, ((row+r)*4)%16, 1)
			if err != nil {
				return nil, err
			}
			copy(y[(row+r)*l.Out+col:], values)
		}
	}
	return y, nil
}
