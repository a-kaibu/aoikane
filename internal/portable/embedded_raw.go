//go:build !(js && wasm) && !aoikane_compact

package portable

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/a-kaibu/aoikane/assets"
)

// Use the same cache identity as the existing source builds.
func embeddedKey() (string, error) {
	h := sha256.New()
	err := fs.WalkDir(assets.FS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		info, err := d.Info()
		if err == nil {
			fmt.Fprintf(h, "%s %d\n", p, info.Size())
		}
		return err
	})
	return hex.EncodeToString(h.Sum(nil))[:16], err
}

func materializeEmbedded(dir string) error {
	return fs.WalkDir(assets.FS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(dir, filepath.FromSlash(p))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := assets.FS.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
}
