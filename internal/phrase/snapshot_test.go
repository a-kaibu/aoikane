package phrase

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/a-kaibu/aoikane/internal/frontend"
)

func testCodec(t *testing.T) *frontend.CP932Codec {
	t.Helper()
	path := filepath.Join("..", "..", "private", "cp932.json")
	if _, err := os.Stat(path); err != nil {
		t.Skip("private CP932 table is not available")
	}
	c, err := frontend.OpenCP932Codec(path)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestLoadSnapshot(t *testing.T) {
	codec := testCodec(t)
	path := filepath.Join("..", "..", "private", "graphs", "c002.jsonl")
	if _, err := os.Stat(path); err != nil {
		t.Skip("probe snapshots are not available")
	}
	snaps, err := ReadSnapshots(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range snaps {
		if s.Nodes == nil {
			continue
		}
		doc, err := s.Load(codec)
		if err != nil {
			t.Fatalf("%s: %v", s.Event, err)
		}
		t.Logf("%s\n%s", s.Event, doc.Describe())
	}
}
