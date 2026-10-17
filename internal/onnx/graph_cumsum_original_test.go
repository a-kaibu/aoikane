package onnx

import (
	"math"
	"testing"
)

func TestOriginalCumSumPhonemeBoundary(t *testing.T) {
	// First 45 durations from an ordinary original-engine utterance. Its
	// frame-expanded features switch phones at frame 775, not frame 776.
	bits := []uint32{
		0x42c80000, 0x422d6a61, 0x43061846, 0x42a4dfd5, 0x425fc622,
		0x42a5728b, 0x41dc6f79, 0x42cbbb04, 0x426443a4, 0x42f6621f,
		0x42af45e2, 0x420157ac, 0x42e93543, 0x42b0bddf, 0x42329151,
		0x42c75202, 0x42c0e75b, 0x42fa5797, 0x42bd36de, 0x4244f582,
		0x429a4ea2, 0x42adfc48, 0x42cd68a1, 0x42b3dad2, 0x42adfdb9,
		0x425b2551, 0x423cb237, 0x42b9a97b, 0x43407e1e, 0x42aad18f,
		0x42aca457, 0x42e2c254, 0x42d21712, 0x42a06a32, 0x42becbbd,
		0x42d6c8cd, 0x42ea67d9, 0x42d0e447, 0x42c80000, 0x41b5bd93,
		0x429c38b8, 0x42cecd77, 0x427f310c, 0x426c0a7e, 0x42a34eb8,
	}
	x := Value{Kind: 1, Shape: []int64{1, 1, int64(len(bits))}, F: make([]float32, len(bits))}
	for i, b := range bits {
		x.F[i] = math.Float32frombits(b)
	}
	axis := Value{Kind: 7, I: []int64{2}}
	for _, original := range []bool{false, true} {
		y, err := executeNode(graphNode{op: "CumSum"}, []Value{x, axis}, original)
		if err != nil {
			t.Fatal(err)
		}
		wantBits, wantFrame := uint32(0x45725800), 776
		if original {
			wantBits, wantFrame = 0x457257ff, 775
		}
		sum := y[0].F[len(bits)-1]
		frame := int(math.RoundToEven(float64(float32(sum*200) / 1000)))
		if math.Float32bits(sum) != wantBits || frame != wantFrame {
			t.Fatalf("original=%t: cumulative=%08x frame=%d, want %08x frame=%d", original, math.Float32bits(sum), frame, wantBits, wantFrame)
		}
	}
}
