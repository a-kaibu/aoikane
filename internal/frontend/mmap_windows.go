//go:build windows

package frontend

import (
	"os"
	"syscall"
	"unsafe"
)

// mapFile maps a dictionary file read-only for the life of the process, so
// start-up neither copies it nor touches pages that analysis never reads.
func mapFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := info.Size()
	if size <= 0 || size != int64(int(size)) {
		return os.ReadFile(path)
	}
	h, err := syscall.CreateFileMapping(syscall.Handle(f.Fd()), nil, syscall.PAGE_READONLY, uint32(size>>32), uint32(size), nil)
	if err != nil {
		return os.ReadFile(path)
	}
	defer syscall.CloseHandle(h)
	addr, err := syscall.MapViewOfFile(h, syscall.FILE_MAP_READ, 0, 0, uintptr(size))
	if err != nil || addr == 0 {
		return os.ReadFile(path)
	}
	// The view stays mapped for the life of the process.
	return unsafe.Slice((*byte)(*(*unsafe.Pointer)(unsafe.Pointer(&addr))), int(size)), nil
}
