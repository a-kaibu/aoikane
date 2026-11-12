//go:build js && wasm

package onnx

import (
	"math"
	"math/rand/v2"
	"syscall/js"
	"testing"
)

func TestWebKernelGEMM(t *testing.T) {
	if gemmNormalKernel.Type() != js.TypeFunction {
		t.Skip("optional kernel not installed")
	}
	rng := rand.New(rand.NewPCG(71, 72))
	for _, m := range []int{1, 3, 8, 11} {
		for _, n := range []int{1, 3, 5, 64} {
			for _, k := range []int{1, 7, 8, 35} {
				for _, align := range []int{0, 4, 8, 12} {
					for _, beta := range []float32{0, 0.5, 1} {
						lda, ldb, ldc := m+3, k+2, m+1
						a, b, seed := make([]float32, lda*k), make([]float32, ldb*n), make([]float32, ldc*n)
						for _, v := range [][]float32{a, b, seed} {
							for i := range v {
								v[i] = float32(rng.Float64()*2 - 1)
							}
						}
						saveG, saveV := gemmNormalKernel, gemvNormalKernel
						gemmNormalKernel, gemvNormalKernel = js.Undefined(), js.Undefined()
						want, err := MKLGEMMNormalN(a, b, seed, m, n, k, lda, ldb, ldc, align, -0.75, beta)
						gemmNormalKernel, gemvNormalKernel = saveG, saveV
						if err != nil {
							t.Fatal(err)
						}
						got, err := MKLGEMMNormalN(a, b, seed, m, n, k, lda, ldb, ldc, align, -0.75, beta)
						if err != nil {
							t.Fatal(err)
						}
						for i := range got {
							if math.Float32bits(got[i]) != math.Float32bits(want[i]) {
								t.Fatalf("m=%d n=%d k=%d align=%d beta=%g index=%d got=%g want=%g", m, n, k, align, beta, i, got[i], want[i])
							}
						}
					}
				}
			}
		}
	}
}
