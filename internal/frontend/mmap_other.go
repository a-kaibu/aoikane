//go:build !unix && !windows

package frontend

import "os"

func mapFile(path string) ([]byte, error) { return os.ReadFile(path) }
