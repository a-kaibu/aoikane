package frontend

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"testing"
)

func TestDictionaryFeatureIntegrity(t *testing.T) {
	for _, c := range []struct{ name, sha string }{
		{"dict", "7113d70ad31cea09f5098d4788ecf806cc6c9217c841a3e3641bb110227af603"},
		{"dict-kansai", "4c0de255c94437060bc3b97bbdbabc82d0394674cd02ad46bb0ccba1c171ca45"},
	} {
		t.Run(c.name, func(t *testing.T) {
			data, err := readDictionaryFile(filepath.Join("..", "..", "assets", c.name, "sys.fea"))
			if err != nil {
				t.Fatal(err)
			}
			if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != c.sha {
				t.Fatalf("dictionary feature data changed: got %s want %s", got, c.sha)
			}
		})
	}
}
