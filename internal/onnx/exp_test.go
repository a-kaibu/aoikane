package onnx

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand/v2"
	"testing"
)

func TestExpMatchesOriginal(t *testing.T) {
	testUnaryLogExp(t, false)
}

func TestLogMatchesOriginal(t *testing.T) {
	testUnaryLogExp(t, true)
}

func testUnaryLogExp(t *testing.T, logarithm bool) {
	t.Helper()
	r := rand.New(rand.NewPCG(13, 7))
	hash := sha256.New()
	total := 0
	for _, length := range []int{1, 2, 5, 17, 65, 257, 4096} {
		input := make([]float32, length)
		for i := range input {
			input[i] = float32(r.Float64()*240 - 120)
			if logarithm {
				input[i] = float32(math.Exp(float64(input[i]) / 2))
			}
		}
		if length == 4096 && !logarithm {
			for i, x := range []float32{0, math.Float32frombits(0x80000000), float32(math.Inf(-1)), float32(math.Inf(1)), math.Float32frombits(0x7fc00000), -103.9720840454, 88.7762626647950, -88.3762626647949} {
				input[i] = x
			}
		}
		if length == 4096 && logarithm {
			for i, x := range []float32{20, math.Nextafter32(20, 21), 440, 0.01, 0.5, 1, 2, 1e-30} {
				input[i] = x
			}
		}
		values := make([]float32, length)
		if logarithm {
			values = logValues(input)
		} else {
			for i, x := range input {
				values[i] = Exp(x)
			}
		}
		for _, x := range values {
			hash.Write(binary.LittleEndian.AppendUint32(nil, math.Float32bits(x)))
		}
		total += length
	}
	// Independently recorded by verify_exp using original Runtime. Exp includes
	// underflow/overflow and special values; Log covers positive packets/tails.
	want := "85771d37db62794025d7e35e8cbc6e386616d74e8c09b1985ad1d9350013824f"
	if logarithm {
		want = "4c116dff6f31b94413e006479e165f85dd23d0a182a35ed3cdd6e630fd80f01f"
	}
	if got := fmt.Sprintf("%x", hash.Sum(nil)); got != want || total != 4443 {
		t.Fatalf("%d unary values SHA256 %s want %s", total, got, want)
	}
}
