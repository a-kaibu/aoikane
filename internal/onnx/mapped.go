package onnx

import (
	"sync"
	"unsafe"
)

// Read-only model mappings whose contents initializers may alias.
var (
	mappedMu      sync.RWMutex
	mappedRegions [][2]uintptr
)

func addMappedRegion(b []byte) {
	if len(b) == 0 {
		return
	}
	start := uintptr(unsafe.Pointer(unsafe.SliceData(b)))
	mappedMu.Lock()
	mappedRegions = append(mappedRegions, [2]uintptr{start, start + uintptr(len(b))})
	mappedMu.Unlock()
}

// inMappedRegion reports whether p points into a model mapping, which is
// outside the Go heap.
func inMappedRegion(p unsafe.Pointer) bool {
	a := uintptr(p)
	mappedMu.RLock()
	defer mappedMu.RUnlock()
	for _, r := range mappedRegions {
		if a >= r[0] && a < r[1] {
			return true
		}
	}
	return false
}
