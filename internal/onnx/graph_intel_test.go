package onnx

import "testing"

// Verify graph selection using ordinary DLL outputs, including exceptional values.
func TestOriginalGraphIntelUnary(t *testing.T) {
	t.Run("Exp", func(t *testing.T) {
		testOriginalAVX512Unary(t, "exp", "0x31640", 21810, 45, "487935e2fe81b176d7ec628f6073f74fbc88efe6f3c50e7c28aeb18b03baf13c", MKLExpAVX512, "Exp")
	})
	t.Run("Log", func(t *testing.T) {
		testOriginalAVX512Unary(t, "log", "0x31b10", 33548, 40, "bedd876a754895a4a03f237253b41d4c1d2a66583e704bf2a84bd72afa39a234", MKLLogAVX512, "Log")
	})
	t.Run("Sin", func(t *testing.T) {
		testOriginalAVX512Unary(t, "sin", "0x329a0", 138474, 45, "787a40a573b7247117c5e25f8a68cdf98bd430dc305f99301d24be4f2418c6be", MKLSinAVX512, "Sin")
	})
	t.Run("Tanh", func(t *testing.T) {
		testOriginalAVX512Unary(t, "tanh", "0x33020", 139794, 45, "5bb96f3b255140ed4150542efa7083895c2090eac44c3610dc7a19c10b153f78", MKLTanhAVX512, "Tanh")
	})
}
