package onnx

import "fmt"

// RunOriginalEmbedding executes the audited original six-thread MKL path for
// FeatNet's art (32->256), nima (10->256), and niab (5->256) projections. Original
// operands are 64-byte aligned. The partition is numerical and runs identically
// with any Go worker count. Other Linear shapes remain unsupported here.
func (l *Linear) RunOriginalEmbedding(input []float32, rows int) ([]float32, error) {
	return l.runOriginalEmbedding(input, rows, 6)
}

func (l *Linear) runOriginalEmbedding(input []float32, rows, threads int) ([]float32, error) {
	if l.Out != 256 || (l.In != 5 && l.In != 10 && l.In != 32) || rows < 2 || len(input)%l.In != 0 || len(input)/l.In != rows || len(l.weight) != l.In*l.Out {
		return nil, fmt.Errorf("unsupported original embedding shape")
	}
	if rows > int(^uint(0)>>1)/l.Out {
		return nil, fmt.Errorf("original embedding output too large")
	}
	weight := make([]float32, len(l.weight))
	for col := 0; col < l.Out; col++ {
		for k := 0; k < l.In; k++ {
			weight[col*l.In+k] = l.weight[k*l.Out+col]
		}
	}
	output := make([]float32, rows*l.Out)
	var transposed []float32
	if l.In == 32 {
		// The original art operand has strides [rows*32,1,rows]. Keep that
		// physical layout for its T,T GEMM and strided GEMV, even though Go
		// graph values are supplied as contiguous logical rows.
		transposed = make([]float32, len(input))
		for row := 0; row < rows; row++ {
			for k := 0; k < l.In; k++ {
				transposed[k*rows+row] = input[row*l.In+k]
			}
		}
	}
	// The original planner (0x136380) splits the larger output dimension,
	// choosing the sequence dimension on equality. Workers 0x136b90 and
	// 0x136e10 use the same remainder allocation.
	axis := l.Out
	if rows >= l.Out {
		axis = rows
	}
	for _, part := range originalMKLPartitions(axis, threads) {
		start, width, firstRow, countRows := part[0], part[1], 0, rows
		if rows >= l.Out {
			start, width, firstRow, countRows = 0, l.Out, part[0], part[1]
		}
		a := weight[start*l.In : (start+width)*l.In]
		b := input[firstRow*l.In : (firstRow+countRows)*l.In]
		if l.In == 5 || countRows < 8 {
			// With fewer than eight sequence rows, the original planner calls
			// generic GEMM for the entire partition, rather than GEMV tails.
			var y []float32
			var err error
			if l.In == 32 {
				y, err = MKLGEMMTransposeT(a, transposed[firstRow:], width, countRows, l.In, l.In, rows, 1)
			} else {
				y, err = MKLGEMMTransposeN(a, b, width, countRows, l.In, l.In, l.In, (firstRow*l.In*4)%16, 1)
			}
			if err != nil {
				return nil, err
			}
			for row := 0; row < countRows; row++ {
				copy(output[(firstRow+row)*l.Out+start:], y[row*width:(row+1)*width])
			}
			continue
		}
		packedRows, packedColumns := countRows&^3, width&^3
		for row := 0; row < packedRows; row++ {
			for col := 0; col < packedColumns; col++ {
				sum := float32(0)
				for k := 0; k < l.In; k++ {
					sum += float32(input[(firstRow+row)*l.In+k] * a[col*l.In+k])
				}
				output[(firstRow+row)*l.Out+start+col] = sum
			}
		}
		if packedRows > 0 && packedColumns != width {
			tail := width - packedColumns
			var y []float32
			var err error
			if l.In == 32 {
				y, err = MKLGEMMTransposeT(a[packedColumns*l.In:], transposed[firstRow:], tail, packedRows, l.In, l.In, rows, 1)
			} else {
				y, err = MKLGEMMTransposeN(a[packedColumns*l.In:], b[:packedRows*l.In], tail, packedRows, l.In, l.In, l.In, (firstRow*l.In*4)%16, 1)
			}
			if err != nil {
				return nil, err
			}
			for row := 0; row < packedRows; row++ {
				copy(output[(firstRow+row)*l.Out+start+packedColumns:], y[row*tail:(row+1)*tail])
			}
		}
		for row := packedRows; row < countRows; row++ {
			x, increment, alignment := b[row*l.In:(row+1)*l.In], 1, ((firstRow+row)*l.In*4)%16
			if l.In == 32 {
				x, increment, alignment = transposed[firstRow+row:], rows, ((firstRow+row)*4)%16
			}
			y, err := MKLGEMVTranspose(a, x, l.In, width, l.In, increment, (start*l.In*4)%16, alignment, 1)
			if err != nil {
				return nil, err
			}
			copy(output[(firstRow+row)*l.Out+start:], y)
		}
	}
	return output, nil
}

func originalMKLPartitions(length int, threads ...int) [][2]int {
	if originalThreadCount(threads) == 1 {
		return [][2]int{{0, length}}
	}
	step := length / 6
	count := 6
	if length%6 > 1 {
		step++
		count = (length + step - 1) / step
	}
	parts := make([][2]int, 0, count)
	for i := 0; i < count; i++ {
		size := step
		if i == count-1 {
			size = length - i*step
		}
		parts = append(parts, [2]int{i * step, size})
	}
	return parts
}
