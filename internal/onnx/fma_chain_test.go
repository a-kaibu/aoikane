package onnx

import (
	"math"
	"math/rand"
	"testing"
)

func TestFMAChainsMatchScalar(t *testing.T) {
	if !useAVX2FMA {
		t.Skip("no AVX2/FMA")
	}
	rng := rand.New(rand.NewSource(1))
	val := func() float32 {
		switch rng.Intn(10) {
		case 0:
			return 0
		case 1:
			return float32(rng.NormFloat64() * 1e-38)
		case 2:
			return math.Float32frombits(rng.Uint32() & 0x807fffff) // subnormal
		}
		return float32(rng.NormFloat64() * math.Pow(2, float64(rng.Intn(20)-10)))
	}
	for trial := 0; trial < 300; trial++ {
		rows := []int{8, 16, 24, 32, 40}[rng.Intn(5)]
		steps := 1 + rng.Intn(70)
		wstep := 1 + rng.Intn(3)
		x := make([]float32, 400)
		for i := range x {
			x[i] = val()
		}
		xoff := make([]int32, steps)
		for i := range xoff {
			xoff[i] = int32(rng.Intn(len(x) - rows))
		}
		var w [chainCols][]float32
		for c := range w {
			w[c] = make([]float32, steps*wstep)
			for i := range w[c] {
				w[c][i] = val()
			}
		}
		for _, fused := range []bool{true, false} {
			acc := make([]float32, chainCols*rows)
			for i := range acc {
				acc[i] = val()
			}
			want := append([]float32(nil), acc...)
			useAVX2FMA = false
			fmaChains(want, rows, x, 0, xoff, w, wstep, fused)
			useAVX2FMA = true
			fmaChains(acc, rows, x, 0, xoff, w, wstep, fused)
			for i := range acc {
				if math.Float32bits(acc[i]) != math.Float32bits(want[i]) {
					t.Fatalf("trial %d fused=%v lane %d: %x != %x", trial, fused, i, math.Float32bits(acc[i]), math.Float32bits(want[i]))
				}
			}
		}
	}
}

func BenchmarkFMA32(b *testing.B) {
	a := float32(0.1)
	for i := 0; i < b.N; i++ {
		a = fma32(a, 0.999, 0.001)
	}
	_ = a
}

func TestFMA32FastPathMatchesExact(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	v := func() float32 {
		if rng.Intn(4) == 0 {
			return math.Float32frombits(rng.Uint32())
		}
		return float32(rng.NormFloat64() * math.Pow(2, float64(rng.Intn(60)-30)))
	}
	for i := 0; i < 2000000; i++ {
		a, b, c := v(), v(), v()
		got, want := fma32(a, b, c), fma32Slow(a, b, c)
		if math.Float32bits(got) != math.Float32bits(want) && !(got != got && want != want) {
			t.Fatalf("fma32(%g, %g, %g) = %g, want %g", a, b, c, got, want)
		}
	}
}

func TestFMAChainColMatchesScalar(t *testing.T) {
	if !useAVX2FMA {
		t.Skip("no AVX2/FMA")
	}
	rng := rand.New(rand.NewSource(3))
	for trial := 0; trial < 200; trial++ {
		rows := 8 * (1 + rng.Intn(12))
		steps := 1 + rng.Intn(50)
		x := make([]float32, 600)
		for i := range x {
			x[i] = float32(rng.NormFloat64())
		}
		xoff := make([]int32, steps)
		for i := range xoff {
			xoff[i] = int32(rng.Intn(len(x) - rows))
		}
		w := make([]float32, steps*2)
		for i := range w {
			w[i] = float32(rng.NormFloat64())
		}
		acc := make([]float32, rows)
		for i := range acc {
			acc[i] = float32(rng.NormFloat64())
		}
		want := append([]float32(nil), acc...)
		useAVX2FMA = false
		fmaChainCol(want, x, 0, xoff, w, 2)
		useAVX2FMA = true
		fmaChainCol(acc, x, 0, xoff, w, 2)
		for i := range acc {
			if math.Float32bits(acc[i]) != math.Float32bits(want[i]) {
				t.Fatalf("trial %d lane %d differs", trial, i)
			}
		}
	}
}

func TestFMAChainsIndexedMatchesScalar(t *testing.T) {
	if !useAVX2FMA {
		t.Skip("no AVX2/FMA")
	}
	rng := rand.New(rand.NewSource(5))
	for trial := 0; trial < 200; trial++ {
		rows := []int{8, 16, 24, 40}[rng.Intn(4)]
		steps := 1 + rng.Intn(60)
		x := make([]float32, 500)
		for i := range x {
			x[i] = float32(rng.NormFloat64())
		}
		xoff, woff := make([]int32, steps), make([]int32, steps)
		for i := range xoff {
			xoff[i] = int32(rng.Intn(len(x) - rows))
			woff[i] = int32(rng.Intn(100))
		}
		var w [chainCols][]float32
		for c := range w {
			w[c] = make([]float32, 100)
			for i := range w[c] {
				w[c][i] = float32(rng.NormFloat64())
			}
		}
		acc := make([]float32, chainCols*rows)
		for i := range acc {
			acc[i] = float32(rng.NormFloat64())
		}
		want := append([]float32(nil), acc...)
		useAVX2FMA = false
		fmaChainsIndexed(want, rows, x, 0, xoff, w, woff)
		useAVX2FMA = true
		fmaChainsIndexed(acc, rows, x, 0, xoff, w, woff)
		for i := range acc {
			if math.Float32bits(acc[i]) != math.Float32bits(want[i]) {
				t.Fatalf("trial %d lane %d differs", trial, i)
			}
		}
	}
}
