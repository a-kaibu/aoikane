package portable

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// parsedCacheVersion changes whenever a cached type changes shape.
const parsedCacheVersion = 1

// cachedRules returns the parsed form of the rule file at path, reading a
// gob snapshot from the user cache when one exists for this exact file
// (path, size and modification time). Parsing validates the text; the
// snapshot is written only from a successful parse. Types cached here must
// have only exported fields, which a round-trip test checks.
func cachedRules[T any](path, kind string, parse func(io.Reader) (*T, error)) (*T, error) {
	file := parsedCachePath(path, kind)
	if file != "" {
		if b, err := os.ReadFile(file); err == nil {
			v := new(T)
			if gob.NewDecoder(bytes.NewReader(b)).Decode(v) == nil {
				return v, nil
			}
		}
	}
	v, err := openRules(path, parse)
	if err != nil || file == "" {
		return v, err
	}
	var buf bytes.Buffer
	if gob.NewEncoder(&buf).Encode(v) == nil {
		writeAtomic(file, buf.Bytes())
	}
	return v, nil
}

func parsedCachePath(path, kind string) string {
	if os.Getenv("AOIKANE_NO_CACHE") != "" {
		return ""
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	info, err := os.Stat(abs)
	if err != nil {
		return ""
	}
	base, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	h := sha256.Sum256([]byte(fmt.Sprintf("%d\x00%s\x00%s\x00%d\x00%d", parsedCacheVersion, kind, abs, info.Size(), info.ModTime().UnixNano())))
	return filepath.Join(base, "aoikane", "parsed", hex.EncodeToString(h[:16])+".gob")
}

func writeAtomic(file string, b []byte) {
	dir := filepath.Dir(file)
	if os.MkdirAll(dir, 0o755) != nil {
		return
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return
	}
	_, werr := tmp.Write(b)
	cerr := tmp.Close()
	if werr != nil || cerr != nil || os.Rename(tmp.Name(), file) != nil {
		os.Remove(tmp.Name())
	}
}
