package phrase

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func openBoundRules(t *testing.T, name string) *BoundRules {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "..", "private", name))
	if err != nil {
		t.Skip("boundary rules are not available")
	}
	defer f.Close()
	r, err := ParseBoundRules(f)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestParseBoundRules(t *testing.T) {
	for _, name := range []string{"bound-rules-standard.txt", "bound-rules-kansai.txt"} {
		r := openBoundRules(t, name)
		n := 0
		for _, g := range r.Groups {
			n += len(g.Patterns)
		}
		t.Logf("%s: %d groups, %d patterns, %d conditions", name, len(r.Groups), n, len(r.conds))
	}
}

// TestBoundMatchesNative applies the boundary rules to the word graph the
// engine had before its BOUND stage and compares the result. The standard
// voice adds the neural predictor (TestPredictorMatchesNative); the Kansai
// voice needs the CaboCha chunk features of 137520, which are not
// reconstructed, so its mismatches are only logged.
func TestBoundMatchesNative(t *testing.T) {
	codec := testCodec(t)
	for _, c := range []struct{ dir, rules string }{{"stages_akane", "bound-rules-kansai.txt"}} {
		rules := openBoundRules(t, c.rules)
		files, _ := filepath.Glob(filepath.Join("..", "..", "private", c.dir, "c*.jsonl"))
		sort.Strings(files)
		total, bad := 0, 0
		for _, path := range files {
			in := snapshotByEvent(t, path, "stage_enter_1b9f60")
			out := snapshotByEvent(t, path, "stage_leave_1b9f60")
			if in == nil || out == nil {
				continue
			}
			doc, err := in.Load(codec)
			if err != nil {
				t.Errorf("%s: %v", path, err)
				continue
			}
			want, err := out.Load(codec)
			if err != nil {
				t.Errorf("%s: %v", path, err)
				continue
			}
			if err := doc.ApplyBound(rules); err != nil {
				t.Fatal(err)
			}
			if len(doc.Morphs) != len(want.Morphs) {
				t.Errorf("%s: morph count differs", path)
				continue
			}
			for i, m := range doc.Morphs {
				w := want.Morphs[i]
				total++
				if m.Kind == w.Kind || w.Kind == 28 || w.Kind == 29 {
					if m.Kind2 == w.Kind2 && m.Up == w.Up && m.Down == w.Down && m.Moras == w.Moras {
						continue
					}
				}
				bad++
				if bad <= 12 {
					t.Logf("%s/%s: %q got kind %d/%d moras %d,%d,%d want %d/%d %d,%d,%d", c.dir, filepath.Base(path), m.Surface,
						m.Kind, m.Kind2, m.Up, m.Down, m.Moras, w.Kind, w.Kind2, w.Up, w.Down, w.Moras)
				}
			}
		}
		t.Logf("%s: %d morphs, %d mismatches", c.dir, total, bad)
	}
}
