package frontend

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestDictionaryFileValidation(t *testing.T) {
	payload := []byte("dictionary feature bytes")
	var b bytes.Buffer
	g := gzip.NewWriter(&b)
	if _, err := g.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
	packed := append([]byte(nil), b.Bytes()...)
	for _, c := range []struct {
		name string
		data []byte
		good bool
	}{
		{"valid", packed, true}, {"header", []byte("bad gzip"), false}, {"truncated", packed[:len(packed)-4], false},
		{"checksum", func() []byte { v := append([]byte(nil), packed...); v[len(v)-8] ^= 1; return v }(), false},
	} {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "sys.fea")
			if err := os.WriteFile(path+".gz", c.data, 0600); err != nil {
				t.Fatal(err)
			}
			got, err := readDictionaryFile(path)
			if c.good {
				if err != nil || !bytes.Equal(got, payload) {
					t.Fatalf("got=%q err=%v", got, err)
				}
			} else if err == nil {
				t.Fatal("corrupt compressed dictionary accepted")
			}
		})
	}
	t.Run("plain preferred", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "sys.fea")
		if err := os.WriteFile(path, payload, 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path+".gz", []byte("invalid"), 0600); err != nil {
			t.Fatal(err)
		}
		got, err := readDictionaryFile(path)
		if err != nil || !bytes.Equal(got, payload) {
			t.Fatalf("plain file priority: %q %v", got, err)
		}
	})
	t.Run("missing", func(t *testing.T) {
		_, err := readDictionaryFile(filepath.Join(t.TempDir(), "absent"))
		if !os.IsNotExist(err) {
			t.Fatalf("expected original not-exist error: %v", err)
		}
	})
}
