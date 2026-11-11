//go:build !unix

package frontend

import "os"

// mapFile reads the file. Windows keeps a mapped file locked until it is
// unmapped, which would block deleting or replacing dictionaries, so only
// unix systems map dictionaries.
func mapFile(path string) ([]byte, error) { return os.ReadFile(path) }
