// Package assetfs resolves engine asset reads. Browser builds register the
// files fetched by JavaScript before opening the engine; everywhere else the
// operating system remains the source of the assets.
package assetfs

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sync"
)

var (
	mu    sync.RWMutex
	files = map[string][]byte{}
)

// Put registers an immutable asset. A later Put with the same name replaces
// it. Callers must not modify b after registration.
func Put(name string, b []byte) {
	key := normalize(name)
	mu.Lock()
	files[key] = b
	mu.Unlock()
}

// Lookup returns a registered asset without touching the filesystem.
func Lookup(name string) ([]byte, bool) {
	key := normalize(name)
	mu.RLock()
	b, ok := files[key]
	mu.RUnlock()
	return b, ok
}

// ReadFile returns a registered asset, or reads it from the filesystem when
// no browser asset has been registered. Registered data is shared and must be
// treated as read-only.
func ReadFile(name string) ([]byte, error) {
	if b, ok := Lookup(name); ok {
		return b, nil
	}
	if runtime.GOOS == "js" {
		return nil, missing("open", name)
	}
	return os.ReadFile(name)
}

// Open opens a registered asset for streaming, or opens the filesystem path
// when it has not been registered.
func Open(name string) (io.ReadCloser, error) {
	if b, ok := Lookup(name); ok {
		return io.NopCloser(bytes.NewReader(b)), nil
	}
	if runtime.GOOS == "js" {
		return nil, missing("open", name)
	}
	return os.Open(name)
}

// Remove drops a registered asset.
func Remove(name string) {
	key := normalize(name)
	mu.Lock()
	delete(files, key)
	mu.Unlock()
}

// NotExist reports whether err means the asset is missing.
func NotExist(err error) bool { return errors.Is(err, fs.ErrNotExist) }

func missing(op, name string) error {
	return &fs.PathError{Op: op, Path: name, Err: fs.ErrNotExist}
}

func normalize(name string) string {
	return path.Clean(filepath.ToSlash(name))
}
