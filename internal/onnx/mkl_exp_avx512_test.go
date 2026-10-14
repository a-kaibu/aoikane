package onnx

import "testing"

func TestMKLExpAVX512Original(t *testing.T) {
	testOriginalAVX512Unary(t, "exp", "0x31640", 21810, 45, "487935e2fe81b176d7ec628f6073f74fbc88efe6f3c50e7c28aeb18b03baf13c", MKLExpAVX512)
}
