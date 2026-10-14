package onnx

import "testing"

func TestMKLSinAVX512Original(t *testing.T) {
	testOriginalAVX512Unary(t, "sin", "0x329a0", 138474, 45, "787a40a573b7247117c5e25f8a68cdf98bd430dc305f99301d24be4f2418c6be", MKLSinAVX512)
}
