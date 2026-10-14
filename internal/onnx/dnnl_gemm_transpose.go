package onnx

// DNNLGEMMNormalT reproduces the original oneDNN AVX2 N,T worker. Logical
// B stores an n-by-k column-major matrix. The original driver reduces K in
// 96-element blocks, compared with 256-element blocks in the N,N path.
func DNNLGEMMNormalT(a, b, initial []float32, m, n, k, lda, ldb, ldc int, beta float32) ([]float32, error) {
	return dnnlGEMM(a, b, initial, m, n, k, lda, ldb, ldc, beta, true)
}
