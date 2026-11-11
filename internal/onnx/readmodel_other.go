//go:build !unix

package onnx

import "os"

const canMapWeights = false

func readModel(path string) ([]byte, func(), error) {
	b, err := os.ReadFile(path)
	return b, func() {}, err
}
