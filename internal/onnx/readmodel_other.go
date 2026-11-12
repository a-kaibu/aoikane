//go:build !unix

package onnx

import "github.com/a-kaibu/aoikane/internal/assetfs"

const canMapWeights = false

func readModel(path string) ([]byte, func(), error) {
	b, err := assetfs.ReadFile(path)
	return b, func() {}, err
}
