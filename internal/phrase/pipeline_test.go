package phrase

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/a-kaibu/aoikane/internal/cabocha"
	"github.com/a-kaibu/aoikane/internal/frontend"
	"github.com/a-kaibu/aoikane/internal/numeral"
)

func openFrontend(t *testing.T) *Frontend {
	t.Helper()
	private := filepath.Join("..", "..", "private")
	if _, err := os.Stat(filepath.Join(private, "frontend-data", "sys.dic")); err != nil {
		t.Skip("standard dictionary is not available")
	}
	analyzer, err := frontend.OpenAnalyzer(filepath.Join(private, "frontend-data"))
	if err != nil {
		t.Fatal(err)
	}
	morphology, err := frontend.OpenMorphology(filepath.Join(private, "morphology.json"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(filepath.Join(private, "reading-rules-utf8.txt"))
	if err != nil {
		t.Skip("reading rules are not available")
	}
	defer f.Close()
	rules, err := frontend.ParseReadingRules(f)
	if err != nil {
		t.Fatal(err)
	}
	codec, err := frontend.OpenCP932Codec(filepath.Join(private, "cp932.json"))
	if err != nil {
		t.Fatal(err)
	}
	nf, err := os.Open(filepath.Join("..", "..", "assets", "rules", "bancharcx-standard.txt"))
	if err != nil {
		t.Fatal(err)
	}
	defer nf.Close()
	numerals, err := numeral.ParseRules(nf)
	if err != nil {
		t.Fatal(err)
	}
	uk, err := frontend.OpenUnknownKanaRules(filepath.Join("..", "..", "assets", "rules", "unknown-kana.json"))
	if err != nil {
		t.Fatal(err)
	}
	ua, err := frontend.OpenUnknownAccentRules(filepath.Join("..", "..", "assets", "rules", "unknown-accent.json"))
	if err != nil {
		t.Fatal(err)
	}
	var chunker *cabocha.CRF
	if m, err := cabocha.LoadCRF(filepath.Join(private, "cabocha", "standard-chunk.ipa.model")); err == nil {
		chunker = m
	}
	english, err := frontend.OpenEnglishReader(filepath.Join("..", "..", "assets", "rules"), codec, ua)
	if err != nil {
		t.Fatal(err)
	}
	return &Frontend{Analyzer: analyzer, Morphology: morphology, Reading: rules, Codec: codec, Numerals: numerals, UnknownKana: uk, UnknownAccent: ua, Chunker: chunker, English: english}
}

// multiSentence reports an input whose sentence-final marks continue after the
// first one; the engine's caller splits those before the frontend runs.
func multiSentence(text string) bool {
	rs := []rune(text)
	for i, r := range rs {
		if strings.ContainsRune("。！？!?", r) && i < len(rs)-1 {
			return true
		}
	}
	return false
}

func compareGraphs(got, want *Document) string {
	if len(got.Morphs) != len(want.Morphs) {
		return "morph count differs"
	}
	for i, g := range got.Morphs {
		w := want.Morphs[i]
		if g.Surface != w.Surface || g.Base != w.Base || g.POS != w.POS || g.POS2 != w.POS2 || g.POS3 != w.POS3 ||
			g.Selected != w.Selected || len(g.Cands) != len(w.Cands) {
			return "morph " + g.Surface + " differs: " + describeMorph(g) + " vs " + describeMorph(w)
		}
		for j := range g.Cands {
			if g.Cands[j] != w.Cands[j] {
				return "candidate of " + g.Surface + " differs: " + describeMorph(g) + " vs " + describeMorph(w)
			}
		}
		gc, wc := g.ExpFirst, w.ExpFirst
		for gc != nil && wc != nil {
			if len(gc.Cands) != len(wc.Cands) || (len(gc.Cands) > 0 && gc.Cands[0] != wc.Cands[0]) {
				return "chain of " + g.Surface + " differs: " + describeMorph(gc) + " vs " + describeMorph(wc)
			}
			if gc == g.ExpLast || wc == w.ExpLast {
				break
			}
			gc, wc = gc.Next, wc.Next
		}
		if (g.ExpFirst == nil) != (w.ExpFirst == nil) {
			return "chain of " + g.Surface + " is missing or unexpected"
		}
	}
	return ""
}

func describeMorph(m *Morph) string {
	return m.Surface + "/" + m.Base + " " + string(rune('0')) + " pos=" + hex32(m.POS) + "," + hex32(m.POS2) + "," + hex32(m.POS3) + " sel=" + itoa(int(m.Selected)) + " " + candString(m)
}

func hex32(v uint32) string {
	const digits = "0123456789abcdef"
	out := []byte("0x")
	started := false
	for s := 28; s >= 0; s -= 4 {
		d := (v >> uint(s)) & 15
		if d != 0 || started || s == 0 {
			started = true
			out = append(out, digits[d])
		}
	}
	return string(out)
}

func itoa(v int) string { return string(append([]byte{}, []byte(fmtInt(v))...)) }

func fmtInt(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b []byte
	for v > 0 {
		b = append([]byte{byte('0' + v%10)}, b...)
		v /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

func candString(m *Morph) string {
	s := ""
	for _, c := range m.Cands {
		s += "[" + c.Reading + "," + c.Pronunciation + "," + c.Accent + "," + c.Connection + "]"
	}
	return s
}
