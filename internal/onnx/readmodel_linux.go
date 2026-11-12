package onnx

import "syscall"

// Prefault model pages on Linux to keep page faults out of inference.
const modelMapFlags = syscall.MAP_SHARED | syscall.MAP_POPULATE
