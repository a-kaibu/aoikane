//go:build goexperiment.simd && js && wasm

package onnx

import (
	"fmt"
	"math"
	"math/rand/v2"
	"syscall/js"
	"testing"
)

func TestWasmSIMDChains(t *testing.T) {
	helperFused := js.Global().Get("__aoikaneLSTMGatesKernel").Type() == js.TypeFunction
	for _, rows := range []int{4, 5, 7, 8, 9, 12, 16, 24, 255, 256, 257} {
		for _, step := range []int{0, 1, 3} {
			for _, fused := range []bool{false, true} {
				t.Run(fmt.Sprintf("rows=%d/step=%d/fused=%v", rows, step, fused), func(t *testing.T) {
					rng := rand.New(rand.NewPCG(42, 7))
					random := func(n int) []float32 {
						v := make([]float32, n)
						for i := range v {
							v[i] = float32(rng.Float64()*2 - 1)
						}
						return v
					}
					x := random(rows + 40)
					xoff := []int32{-3, 9, 0, 17, 8, 2, 19}
					woff := []int32{5, 2, 8, 0, 10, 1, 4}
					var w [chainCols][]float32
					for c := range w {
						w[c] = random(24)
					}
					initial := random(chainCols * rows)
					for _, indexed := range []bool{false, true} {
						got := append([]float32(nil), initial...)
						ok := false
						if indexed {
							ok = fmaChainsIndexedSIMD(got, rows, x, 3, xoff, w, woff)
						} else {
							ok = fmaChainsSIMD(got, rows, x, 3, xoff, w, step, fused)
						}
						if !ok {
							t.Fatal("kernel rejected valid operands")
						}
						for c := range w {
							for r := 0; r < rows; r++ {
								want := initial[c*rows+r]
								for j, o := range xoff {
									wi := j * step
									if indexed {
										wi = int(woff[j])
									}
									v, weight := x[3+int(o)+r], w[c][wi]
									if (r >= rows&^3 || helperFused) && (indexed || fused) {
										want = fma32(v, weight, want)
									} else {
										want = float32(want + float32(v*weight))
									}
								}
								if math.Float32bits(got[c*rows+r]) != math.Float32bits(want) {
									t.Fatalf("indexed=%v col=%d row=%d: got %g, want %g", indexed, c, r, got[c*rows+r], want)
								}
							}
						}
					}
				})
			}
		}
	}
}

func TestWasmSIMDInvalidOperands(t *testing.T) {
	var w [chainCols][]float32
	for c := range w {
		w[c] = []float32{1, 2}
	}
	x := make([]float32, 16)
	for _, tc := range []struct {
		name             string
		base, rows, step int
		xoff, woff       []int32
	}{
		{"negative input", 0, 8, 1, []int32{-1}, []int32{0}},
		{"input end", 9, 8, 1, []int32{0}, []int32{0}},
		{"overflow base", math.MaxInt, 8, 1, []int32{0, 1}, []int32{0, 1}},
		{"short accumulator", 0, 9, 1, []int32{0}, []int32{0}},
		{"short weight", 0, 8, 3, []int32{0, 1}, []int32{0, 3}},
		{"negative weight", 0, 8, -1, []int32{0, 1}, []int32{0, -1}},
		{"no steps", 0, 8, 1, nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			acc := make([]float32, 32)
			if fmaChainsSIMD(acc, tc.rows, x, tc.base, tc.xoff, w, tc.step, true) {
				t.Fatal("accepted invalid strided operands")
			}
			if fmaChainsIndexedSIMD(acc, tc.rows, x, tc.base, tc.xoff, w, tc.woff) {
				t.Fatal("accepted invalid indexed operands")
			}
			for _, v := range acc {
				if v != 0 {
					t.Fatal("modified accumulator before validation")
				}
			}
		})
	}
}

func BenchmarkWasmSIMDChains(b *testing.B) {
	const rows, steps = 256, 512
	x := make([]float32, rows+steps)
	xoff, woff := make([]int32, steps), make([]int32, steps)
	var w [chainCols][]float32
	for c := range w {
		w[c] = make([]float32, steps)
		for j := range w[c] {
			w[c][j] = 0.001
		}
	}
	for i := range x {
		x[i] = 0.01
	}
	for j := range xoff {
		xoff[j], woff[j] = int32(j), int32(steps-1-j)
	}
	acc := make([]float32, chainCols*rows)
	b.Run("strided", func(b *testing.B) {
		for b.Loop() {
			fmaChainsSIMD(acc, rows, x, 0, xoff, w, 1, true)
		}
	})
	b.Run("indexed", func(b *testing.B) {
		for b.Loop() {
			fmaChainsIndexedSIMD(acc, rows, x, 0, xoff, w, woff)
		}
	})
}
