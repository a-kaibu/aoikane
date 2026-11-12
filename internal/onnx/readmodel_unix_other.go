//go:build unix && !linux

package onnx

import "syscall"

const modelMapFlags = syscall.MAP_SHARED
