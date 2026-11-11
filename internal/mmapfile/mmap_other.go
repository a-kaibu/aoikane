//go:build !unix

// Package mmapfile maps read-only asset files for the life of the process.
package mmapfile

import "os"

// Read reads path. Windows locks mapped files until they are unmapped, so
// only unix systems map assets.
func Read(path string) ([]byte, error) { return os.ReadFile(path) }
