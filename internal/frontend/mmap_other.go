//go:build !unix

package frontend

import "github.com/a-kaibu/aoikane/internal/assetfs"

// mapFile reads the file. Windows keeps a mapped file locked until it is
// unmapped, which would block deleting or replacing dictionaries, so only
// unix systems map dictionaries. Browser builds read registered assets.
func mapFile(path string) ([]byte, error) { return assetfs.ReadFile(path) }
