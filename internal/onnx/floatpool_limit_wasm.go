//go:build js && wasm

package onnx

// Browser workers each own a pool. Retaining the native 384 MiB limit in every
// worker would exhaust memory after synthesizing several different lengths.
const pooledMaxBytes = 128 << 20
