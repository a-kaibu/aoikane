//go:build unix

package frontend

import (
	"os"
	"syscall"
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
	b, err := syscall.Mmap(int(f.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return os.ReadFile(path)
	}
	return b, nil
}
