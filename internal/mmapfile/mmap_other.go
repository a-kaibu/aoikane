//go:build !unix

// Package mmapfile maps read-only asset files for the life of the process.
package mmapfile

import "github.com/a-kaibu/aoikane/internal/assetfs"

// Read reads path. Windows locks mapped files until they are unmapped, so
// only unix systems map assets. Browser builds read registered assets.
func Read(path string) ([]byte, error) { return assetfs.ReadFile(path) }
