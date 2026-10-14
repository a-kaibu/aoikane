package onnx

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"math/rand/v2"
	"testing"
)

// Recorded from the bundled ONNX Runtime 1.17.1 FMA3 CPU kernels via
// tools/verify_activations, independently of these Go implementations.
func TestActivationsMatchNativeBits(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	input := make([]float32, 2048)
	for i := range input {
		input[i] = float32(r.Float64()*40 - 20)
	}
	var raw []byte
	for _, f := range []func(float32) float32{Sigmoid, Tanh} {
		for _, x := range input {
			raw = binary.LittleEndian.AppendUint32(raw, math.Float32bits(f(x)))
		}
	}
	const want = "1522f29b8ec4a7d7749b5c774cc0e4b066930746d8351dcf8dae164d99b9a55b"
	if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != want {
		t.Fatalf("4096 native activation bits: got SHA256 %s want %s", got, want)
	}
}

func TestFMA32DoubleRounding(t *testing.T) {
	// The tiny negative addend moves an exact halfway product below the
	// midpoint, but is lost when the sum is first rounded to float64.
	a, b, c := float32(1.0000001192092896), float32(1.5), -float32(math.SmallestNonzeroFloat32)
	got := fma32(a, b, c)
	want := math.Float32frombits(math.Float32bits(1.5) + 1)
	if math.Float32bits(got) != math.Float32bits(want) {
		t.Fatalf("double rounding: got %08x want %08x", math.Float32bits(got), math.Float32bits(want))
	}
	if naive := float32(float64(a)*float64(b) + float64(c)); naive == want {
		t.Fatal("regression input must expose float64 double rounding")
	}
}

func TestFMA32SingleRounding(t *testing.T) {
	r := rand.New(rand.NewPCG(42, 19))
	for i := 0; i < 10000; i++ {
		a, b, c := math.Float32frombits(r.Uint32()), math.Float32frombits(r.Uint32()), math.Float32frombits(r.Uint32())
		if math.IsNaN(float64(a)) || math.IsNaN(float64(b)) || math.IsNaN(float64(c)) || math.IsInf(float64(a), 0) || math.IsInf(float64(b), 0) || math.IsInf(float64(c), 0) {
			continue
		}
		x := new(big.Float).SetPrec(1024).SetFloat64(float64(a))
		x.Mul(x, new(big.Float).SetPrec(1024).SetFloat64(float64(b)))
		x.Add(x, new(big.Float).SetPrec(1024).SetFloat64(float64(c)))
		want, _ := x.Float32()
		got := fma32(a, b, c)
		if math.Float32bits(got) != math.Float32bits(want) {
			t.Fatalf("%g*%g+%g: got %08x want %08x", a, b, c, math.Float32bits(got), math.Float32bits(want))
		}
	}
}
