package onnx

import (
	"math"
	"math/rand"
	"testing"
)

func TestFastConvMatchesReference(t *testing.T) {
	if !useAVX2FMA && !wasmSIMDAvailable {
		t.Skip("no SIMD convolution")
	}
	rng := rand.New(rand.NewSource(1))
	for _, s := range []struct{ in, out, k, d, l, left, right int }{
		{8, 8, 3, 1, 100, 1, 1}, {16, 32, 7, 3, 77, 9, 9}, {5, 4, 1, 1, 24, 0, 0}, {3, 12, 5, 2, 50, 4, 4},
	} {
		c := &Conv1D{In: s.in, Out: s.out, Kernel: s.k, Dilation: s.d, Left: s.left, Right: s.right}
		c.weight = make([]float32, s.out*s.in*s.k)
		c.bias = make([]float32, s.out)
		for i := range c.weight {
			c.weight[i] = rng.Float32()*2 - 1
		}
		for i := range c.bias {
			c.bias[i] = rng.Float32()
		}
		x := make([]float32, s.in*s.l)
		for i := range x {
			x[i] = rng.Float32()*2 - 1
		}
		y, ok := c.runFast(x, s.l)
		if !ok {
			t.Fatalf("%+v not handled", s)
		}
		n := s.left + s.l + s.right - (s.k-1)*s.d
		for o := 0; o < s.out; o++ {
			for f := 0; f < n; f++ {
				want := float64(c.bias[o])
				for ci := 0; ci < s.in; ci++ {
					for k := 0; k < s.k; k++ {
						p := f + k*s.d - s.left
						if p >= 0 && p < s.l {
							want += float64(x[ci*s.l+p]) * float64(c.weight[(o*s.in+ci)*s.k+k])
						}
					}
				}
				if got := float64(y[o*n+f]); math.Abs(got-want) > 1e-4 {
					t.Fatalf("%+v out %d frame %d: got %v want %v", s, o, f, got, want)
				}
			}
		}
	}
}
