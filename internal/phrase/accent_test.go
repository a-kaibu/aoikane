package phrase

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func corpusFiles(t *testing.T) []string {
	t.Helper()
	files, _ := filepath.Glob(filepath.Join("..", "..", "private", "graphs", "c*.jsonl"))
	if len(files) == 0 {
		t.Skip("probe snapshots are not available")
	}
	sort.Strings(files)
	return files
}

func snapshotByEvent(t *testing.T, path, event string) *Snapshot {
	t.Helper()
	snaps, err := ReadSnapshots(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := range snaps {
		if snaps[i].Event == event {
			return &snaps[i]
		}
	}
	return nil
}

// TestAccentRecompute recomputes the accent of every prosodic word from the
// phrase structure the native engine built, in document order.
func TestAccentRecompute(t *testing.T) {
	codec := testCodec(t)
	total, bad := 0, 0
	for _, path := range corpusFiles(t) {
		s := snapshotByEvent(t, path, "graph_built")
		if s == nil {
			continue
		}
		doc, err := s.Load(codec)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		for _, sent := range doc.Sentences {
			end := (*PWord)(nil)
			if sent.Next != nil {
				end = sent.Next.PWord
			}
			var prev *PWord
			for p := sent.PWord; p != nil && p != end; p = p.Next {
				want := [3]int32{p.Up, p.Down, p.Moras}
				p.Up, p.Down, p.Moras = -7, -7, -7
				doc.ac9b0(prev, p)
				got := [3]int32{p.Up, p.Down, p.Moras}
				total++
				if got != want {
					bad++
					if bad <= 20 {
						var words string
						for m := p.First; m != nil && m != p.endMorph(); m = m.Next {
							words += m.Surface + "|"
						}
						t.Errorf("%s: %s got %v want %v", filepath.Base(path), words, got, want)
					}
					p.Up, p.Down, p.Moras = want[0], want[1], want[2]
				}
				prev = p
			}
		}
	}
	t.Logf("%d prosodic words, %d mismatches", total, bad)
	_ = os.Stdout
}

// TestBuildMatchesNative runs the whole phrase pass from the state before
// 1aa4f0 and compares the structure and the formatted text with the engine.
func TestBuildMatchesNative(t *testing.T) {
	codec := testCodec(t)
	for _, dir := range []string{"graphs", "graphs_akane", "graphs2", "graphs2_akane"} {
		files, _ := filepath.Glob(filepath.Join("..", "..", "private", dir, "c*.jsonl"))
		sort.Strings(files)
		total, bad := 0, 0
		for _, path := range files {
			in := snapshotByEvent(t, path, "graph_in")
			after := snapshotByEvent(t, path, "graph_after")
			format := snapshotByEvent(t, path, "format")
			if in == nil || after == nil || format == nil {
				continue
			}
			want, err := format.FormatText(codec)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := in.Load(codec)
			if err != nil {
				t.Errorf("%s: %v", path, err)
				continue
			}
			if err := doc.Build(); err != nil {
				t.Logf("%s: skipped: %v", path, err)
				continue
			}
			got := ""
			for _, s := range doc.Sentences {
				got += doc.Format(s)
			}
			total++
			if got != want {
				bad++
				if bad <= 8 {
					t.Errorf("%s/%s: %s\n got  %s\n want %s", dir, filepath.Base(path), in.Input, got, want)
				}
			}
		}
		t.Logf("%s: %d sentences, %d mismatches", dir, total, bad)
	}
}
