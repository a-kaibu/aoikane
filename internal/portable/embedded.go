//go:build !(js && wasm)

package portable

import (
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
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
		expandDictionaries(dir)
		return dir, nil
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return "", err
	}
	tmp, err := os.MkdirTemp(filepath.Dir(dir), key+".tmp-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	if err := materializeEmbedded(tmp); err != nil {
		return "", err
	}

	expandDictionaries(tmp)
	if err := os.WriteFile(filepath.Join(tmp, ".complete"), nil, 0o644); err != nil {
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

// expandDictionaries stores decompressed copies of the gzip dictionary files
// in the extraction cache, so later starts read them without inflating. The
// dictionary reader prefers the uncompressed file and otherwise falls back to
// the .gz file, so a failed expansion only costs start-up time.
func expandDictionaries(dir string) {
	for _, sub := range []string{"dict", "dict-kansai"} {
		matches, _ := filepath.Glob(filepath.Join(dir, sub, "*.gz"))
		for _, compressed := range matches {
			target := strings.TrimSuffix(compressed, ".gz")
			if _, err := os.Stat(target); err == nil {
				continue
			}
			expandFile(compressed, target)
		}
	}
}

func expandFile(compressed, target string) error {
	in, err := os.Open(compressed)
	if err != nil {
		return err
	}
	defer in.Close()
	r, err := gzip.NewReader(in)
	if err != nil {
		return err
	}
	defer r.Close()
	// A unique temporary name lets concurrent first runs expand safely.
	out, err := os.CreateTemp(filepath.Dir(target), filepath.Base(target)+".*.tmp")
	if err != nil {
		return err
	}
	_, err = io.Copy(out, r)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(out.Name(), target)
	}
	if err != nil {
		os.Remove(out.Name())
	}
	return err
}
