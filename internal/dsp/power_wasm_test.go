//go:build js && wasm

package dsp

import (
	"math"
	"math/rand/v2"
	"testing"
)

func TestWebKernelWindowPower(t *testing.T) {
	rng := rand.New(rand.NewPCG(41, 42))
	for _, frames := range []int{1, 2, 31, 32, 33, 64, 110} {
		wave, window := make([]float32, frames*240), make([]float32, 2048)
		for _, v := range [][]float32{wave, window} {
			for i := range v {
				v[i] = float32(rng.Float64()*2 - 1)
			}
		}
		got, ok := windowPowerSums(wave, window)
		if !ok {
			t.Skip("optional kernel not installed")
		}
		c := Compressor{WindowPower: window, FrameSamples: 240}
		for i, v := range got {
			want := c.originalPowerSum(wave, i, i >= frames/32*32)
			if math.Float32bits(v) != math.Float32bits(want) {
				t.Fatalf("frames=%d index=%d got=%g want=%g", frames, i, v, want)
			}
		}
	}
}
