package onnx

import (
	"fmt"
	"math/bits"
)

// torchMeanAll follows PyTorch 2.0.1 CPU SumKernel.cpp's cascade sum and
// ReduceOps.cpp's sum/divide mean. See PYTORCH-LICENSE. Eight logical lanes
// reproduce the original AVX2 reduction without depending on the host SIMD.
func torchMeanAll(x []float32) (float32, error) {
	if len(x) == 0 || len(x) > 32768 {
		return 0, fmt.Errorf("original global mean requires 1..32768 elements; parallel reduction is not implemented")
	}
	lanes := 1
	if len(x) >= 8 {
		lanes = 8
	}
	vectors := len(x) / lanes
	rows := vectors / 4
	power := max(4, bits.Len(uint(max(1, rows)-1))/4)
	step := 1 << power
	var acc [4][4][8]float32
	i := 0
	for i+step <= rows {
		for j := 0; j < step; j++ {
			for k := 0; k < 4; k++ {
				for lane := 0; lane < lanes; lane++ {
					acc[0][k][lane] = float32(acc[0][k][lane] + x[(i*4+k)*lanes+lane])
				}
			}
			i++
		}
		for level := 1; level < 4; level++ {
			for k := 0; k < 4; k++ {
				for lane := 0; lane < lanes; lane++ {
					acc[level][k][lane] = float32(acc[level][k][lane] + acc[level-1][k][lane])
					acc[level-1][k][lane] = 0
				}
			}
			if i&((step-1)<<(level*power)) != 0 {
				break
			}
		}
	}
	for ; i < rows; i++ {
		for k := 0; k < 4; k++ {
			for lane := 0; lane < lanes; lane++ {
				acc[0][k][lane] = float32(acc[0][k][lane] + x[(i*4+k)*lanes+lane])
			}
		}
	}
	for level := 1; level < 4; level++ {
		for k := 0; k < 4; k++ {
			for lane := 0; lane < lanes; lane++ {
				acc[0][k][lane] = float32(acc[0][k][lane] + acc[level][k][lane])
			}
		}
	}
	for v := rows * 4; v < vectors; v++ {
		for lane := 0; lane < lanes; lane++ {
			acc[0][0][lane] = float32(acc[0][0][lane] + x[v*lanes+lane])
		}
	}
	for k := 1; k < 4; k++ {
		for lane := 0; lane < lanes; lane++ {
			acc[0][0][lane] = float32(acc[0][0][lane] + acc[0][k][lane])
		}
	}
	var sum float32
	if lanes == 1 {
		sum = acc[0][0][0]
	} else {
		for _, v := range x[vectors*lanes:] {
			sum = float32(sum + v)
		}
		for lane := 0; lane < lanes; lane++ {
			sum = float32(sum + acc[0][0][lane])
		}
	}
	return float32(sum / float32(len(x))), nil
}
