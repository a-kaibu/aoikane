package onnx

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"testing"
)

func TestMKLLogAVX512Original(t *testing.T) {
	testOriginalAVX512Unary(t, "log", "0x31b10", 33548, 40, "bedd876a754895a4a03f237253b41d4c1d2a66583e704bf2a84bd72afa39a234", MKLLogAVX512)
}

func TestMKLLogAVX512OriginalAllSubnormal(t *testing.T) {
	// Independent ordinary Intel VML capture of every positive subnormal.
	const want = "89d249fe3d4239a352992c0eae776df6904f2e1d1738c3b3cbd044dfb9497d1d"
	h := sha256.New()
	var buf [4]byte
	for b := uint32(1); b < 0x800000; b++ {
		binary.LittleEndian.PutUint32(buf[:], math.Float32bits(MKLLogAVX512(math.Float32frombits(b))))
		h.Write(buf[:])
	}
	got := fmt.Sprintf("%x", h.Sum(nil))
	if got != want {
		t.Fatalf("original 8388607 positive subnormal outputs: got %s want %s", got, want)
	}
	t.Logf("Compared all 8388607 positive subnormal outputs; SHA256 %s", got)
}
