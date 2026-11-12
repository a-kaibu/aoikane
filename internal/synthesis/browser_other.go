//go:build !(js && wasm)

package synthesis

import "github.com/a-kaibu/aoikane/internal/onnx"

func browserRender(Input, onnx.OriginalKernelProfile) ([]byte, bool) { return nil, false }

func browserVocoderParallelism() int                                            { return 0 }
func browserVocode([]modelTensor, onnx.OriginalKernelProfile) ([]float32, bool) { return nil, false }
