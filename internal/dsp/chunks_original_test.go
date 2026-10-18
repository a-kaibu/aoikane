package dsp

import (
	"math"
	"testing"
)

func TestOriginalChunkFadeRounding(t *testing.T) {
	// Recorded at frame100/cursor2143 in the amd64 render matching the
	// ordinary SDK WAV for "８２百八。" (Aoi). ARM64 previously changed
	// PCM sample23743 by one.
	const cursor = 2143
	x := math.Float32frombits(0xbdbd2a16)
	y := math.Float32frombits(0x3da81b70)
	ramp := linearRamp(2400)
	if math.Float32bits(ramp[cursor]) != 0x3f64ad49 || math.Float32bits(ramp[2399-cursor]) != 0x3dda8b3d {
		t.Fatal("original fade ramp differs")
	}
	fused := float32(math.FMA(float64(x), float64(ramp[cursor]), float64(float32(y*ramp[2399-cursor]))))
	if pcmSample(fused) != -2416 {
		t.Fatal("recorded operands no longer expose the fused-rounding difference")
	}
	wave, previous := make([]float32, 2400), make([]float32, 2400)
	wave[cursor], previous[cursor] = x, y
	got, err := JoinChunk(wave, previous, ChunkOptions{
		FrameSamples: 240, OverlapFrames: 10, Threshold: 0.4, Last: true, FrameBegin: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.AppliedLag != 0 || len(got.PCM) != 2400 {
		t.Fatal("isolated fade geometry differs")
	}
	if got.PCM[cursor] != -2417 {
		t.Fatalf("original fade PCM = %d; want -2417", got.PCM[cursor])
	}
}
