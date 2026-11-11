//go:build unix

package onnx

import (
	"os"
	"syscall"
)

// canMapWeights reports that readModel maps files, so weights may alias them.
const canMapWeights = true

// readModel maps a model file read-only for the life of the process. Large
// aligned FLOAT initializers alias the mapping instead of being copied, so
// start-up does not touch weights twice; they are never written.
func readModel(path string) ([]byte, func(), error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, nil, err
	}
	size := info.Size()
	if size > 0 && size == int64(int(size)) {
		if b, err := syscall.Mmap(int(f.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED|syscall.MAP_POPULATE); err == nil {
			addMappedRegion(b)
			return b, func() {}, nil
		}
	}
	b, err := os.ReadFile(path)
	return b, func() {}, err
}
