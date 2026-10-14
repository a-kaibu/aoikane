package numeral

import (
	"os"
	"path/filepath"
	"testing"
)

func openRules(t *testing.T, name string) *Rules {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "..", "assets", "rules", name))
	if err != nil {
		t.Skip(err)
	}
	defer f.Close()
	r, err := ParseRules(f)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestParseRules(t *testing.T) {
	r := openRules(t, "bancharcx-standard.txt")
	t.Logf("digits %d units %d cons %d Tj %d Tk %d accents %d counters %d specials %d suffix %d signs %d pre %d",
		len(r.Digits), len(r.Units), len(r.Cons), len(r.Tj), len(r.Tk), len(r.Accents), len(r.Counters), len(r.Specials), len(r.Suffixes), len(r.Signs), len(r.Pre))
	if len(r.Tk["一"]) != 30 || len(r.Tj["一"]) != 18 {
		t.Errorf("Tk %d Tj %d", len(r.Tk["一"]), len(r.Tj["一"]))
	}
	if c := r.Counters["台"]; c.Tn != "AAB" || c.Tl != "C3" {
		t.Errorf("counter %+v", c)
	}
	if w := r.BySound["ニセン"]; w.Fall != 2 || w.Moras != 3 {
		t.Errorf("accent word %+v", w)
	}
}
