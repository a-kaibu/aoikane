package portable

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/a-kaibu/aoikane/assets"
)

// OpenEmbedded opens the engine from the data compiled into the binary. The
// model and dictionary files are extracted once to the user cache directory;
// later starts reuse them. Neural inference uses Go only.
func OpenEmbedded() (*Engine, error) {
	dir, err := extractEmbedded()
	if err != nil {
		return nil, err
	}
	return Open(Options{AssetsDir: dir})
}

// embeddedKey identifies the embedded content (names and sizes), so a new
// build extracts into a new directory.
func embeddedKey() (string, error) {
	h := sha256.New()
	err := fs.WalkDir(assets.FS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		fmt.Fprintf(h, "%s %d\n", p, info.Size())
		return nil
	})
	return hex.EncodeToString(h.Sum(nil))[:16], err
}

func extractEmbedded() (string, error) {
	key, err := embeddedKey()
	if err != nil {
		return "", err
	}
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	dir := filepath.Join(base, "aoikane", key)
	done := filepath.Join(dir, ".complete")
	if _, err := os.Stat(done); err == nil {
		return dir, nil
	}
	tmp := dir + ".tmp"
	if err := os.RemoveAll(tmp); err != nil {
		return "", err
	}
	err = fs.WalkDir(assets.FS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(tmp, filepath.FromSlash(p))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := assets.FS.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(filepath.Join(tmp, ".complete"), nil, 0o644); err != nil {
		return "", err
	}
	if err := os.RemoveAll(dir); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, dir); err != nil {
		if _, serr := os.Stat(done); serr == nil {
			return dir, nil // another process finished first
		}
		return "", errors.Join(err)
	}
	return dir, nil
}
