//go:build js && wasm

package onnx

import (
	"math"
	"math/rand/v2"
	"syscall/js"
	"testing"
)

func TestWebKernelGEMV(t *testing.T) {
	if gemvNormalKernel.Type() != js.TypeFunction {
		t.Skip("optional kernel not installed")
	}
	rng := rand.New(rand.NewPCG(1, 2))
	for _, m := range []int{1, 3, 4, 5, 11, 64} {
		for _, n := range []int{1, 7, 8, 9, 35} {
			for _, align := range []int{0, 4, 8, 12} {
				for _, beta := range []float32{0, 1, -0.5} {
					lda := m + 3
					a, x, initial := make([]float32, lda*n), make([]float32, n), make([]float32, m)
					for _, v := range [][]float32{a, x, initial} {
						for i := range v {
							v[i] = float32(rng.Float64()*2 - 1)
						}
					}
					save := gemvNormalKernel
					gemvNormalKernel = js.Undefined()
					want, err := MKLGEMVNormal(a, x, initial, m, n, lda, align, -0.75, beta)
					gemvNormalKernel = save
					if err != nil {
						t.Fatal(err)
					}
					got, err := MKLGEMVNormal(a, x, initial, m, n, lda, align, -0.75, beta)
					if err != nil {
						t.Fatal(err)
					}
					for i := range got {
						if math.Float32bits(got[i]) != math.Float32bits(want[i]) {
							t.Fatalf("m=%d n=%d align=%d beta=%v row=%d got=%g want=%g", m, n, align, beta, i, got[i], want[i])
						}
					}
				}
			}
		}
	}
}

func TestWebKernelProjection(t *testing.T) {
	if projectionKernel.Type() != js.TypeFunction {
		t.Skip("optional kernel not installed")
	}
	rng := rand.New(rand.NewPCG(11, 12))
	for _, out := range []int{1, 60} {
		for _, rows := range []int{60, 61, 75, 123} {
			for _, threads := range []int{1, 6} {
				inner := 256
				if out == 1 {
					inner = 64
				}
				l := &Linear{In: inner, Out: out, weight: make([]float32, inner*out)}
				x := make([]float32, rows*inner)
				for _, v := range [][]float32{l.weight, x} {
					for i := range v {
						v[i] = float32(rng.Float64()*2 - 1)
					}
				}
				save := projectionKernel
				projectionKernel = js.Undefined()
				want, err := l.runOriginalLinear(x, rows, threads)
				projectionKernel = save
				if err != nil {
					t.Fatal(err)
				}
				got, err := l.runOriginalLinear(x, rows, threads)
				if err != nil {
					t.Fatal(err)
				}
				for i := range got {
					if math.Float32bits(got[i]) != math.Float32bits(want[i]) {
						t.Fatalf("out=%d rows=%d threads=%d index=%d got=%g want=%g", out, rows, threads, i, got[i], want[i])
					}
				}
			}
		}
	}
}

func TestWebKernelLSTMGates(t *testing.T) {
	if lstmGatesKernel.Type() != js.TypeFunction {
		t.Skip("fused optional kernel not installed")
	}
	save := FastMath()
	SetFastMath(true)
	defer SetFastMath(save)
	rng := rand.New(rand.NewPCG(21, 22))
	l := &lstmLayer{width: 233, hidden: 64, w: make([]float32, 2*256*233), r: make([]float32, 2*256*64), bias: make([]float32, 512), packing: &lstmPacking{}}
	input, state := make([]float32, 233), make([]float32, 64)
	for _, v := range [][]float32{l.w, l.r, l.bias, input, state} {
		for i := range v {
			v[i] = float32(rng.Float64()*2 - 1)
		}
	}
	for d := 0; d < 2; d++ {
		for round := 0; round < 2; round++ {
			state[0] += 0.125
			got := make([]float32, 256)
			if !fastLSTMGatesBulk(got, input, state, l, d) {
				t.Fatal("valid gates rejected")
			}
			for g := range got {
				var a, b float32
				for k, x := range input {
					a = fma32(x, l.w[(d*256+g)*233+k], a)
				}
				for k, x := range state {
					b = fma32(x, l.r[(d*256+g)*64+k], b)
				}
				want := float32(float32(a+b) + l.bias[d*256+g])
				if math.Float32bits(got[g]) != math.Float32bits(want) {
					t.Fatalf("direction=%d gate=%d got=%g want=%g", d, g, got[g], want)
				}
			}
		}
	}
}

func TestWebKernelIndexedConv(t *testing.T) {
	if indexedConvKernel.Type() != js.TypeFunction || lstmGatesKernel.Type() != js.TypeFunction {
		t.Skip("fused optional kernel not installed")
	}
	save := FastMath()
	SetFastMath(true)
	defer SetFastMath(save)
	rng := rand.New(rand.NewPCG(31, 32))
	for _, frames := range []int{8, 9, 11, 12, 15, 17, 31, 110} {
		for _, stride := range []int{1, 2} {
			for _, dilation := range []int{1, 3} {
				length := (frames-1)*stride + 1
				c := &Conv1D{In: 8, Out: 8, Kernel: 3, Stride: stride, Dilation: dilation, Left: dilation, Right: dilation, weight: make([]float32, 8*8*3), bias: make([]float32, 8)}
				x := make([]float32, 8*length)
				for _, v := range [][]float32{x, c.weight, c.bias} {
					for i := range v {
						v[i] = float32(rng.Float64()*2 - 1)
					}
				}
				got := make([]float32, frames*8)
				c.runOriginalDNNLPadded(x, length, frames, got)
				registerConstant(c.weight)
				packed := make([]float32, frames*8)
				c.runOriginalDNNLPadded(x, length, frames, packed)
				for ch := 0; ch < 8; ch++ {
					for t0 := 0; t0 < frames; t0++ {
						want := c.bias[ch]
						for k := 0; k < 3; k++ {
							pos := t0*stride + k*dilation - c.Left
							for ci := 0; ci < 8; ci++ {
								var value float32
								if pos >= 0 && pos < length {
									value = x[ci*length+pos]
								}
								want = fma32(value, c.weight[(ch*8+ci)*3+k], want)
							}
						}
						if math.Float32bits(got[ch*frames+t0]) != math.Float32bits(want) {
							t.Fatalf("frames=%d stride=%d dilation=%d channel=%d t=%d got=%g want=%g", frames, stride, dilation, ch, t0, got[ch*frames+t0], want)
						}
						if math.Float32bits(packed[ch*frames+t0]) != math.Float32bits(want) {
							t.Fatalf("packed frames=%d channel=%d t=%d got=%g want=%g", frames, ch, t0, packed[ch*frames+t0], want)
						}
					}
				}
			}
		}
	}
}
