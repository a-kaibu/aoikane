//go:build unix

// Package mmapfile maps read-only asset files for the life of the process.
package mmapfile

import (
	"os"
	"syscall"
)

// Read maps path read-only. The mapping is never released, so the result
// stays valid; writing to it faults. Systems without mmap read the file.
func Read(path string) ([]byte, error) {
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
	b, err := syscall.Mmap(int(f.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return os.ReadFile(path)
	}
	return b, nil
}
