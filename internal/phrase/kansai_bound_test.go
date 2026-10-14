package phrase

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

type kansaiBoundState struct {
	Kind, Kind2, Up, Down, Moras int32
	Annotation                   string
}
type kansaiBoundCase struct {
	Text   string
	Input  []*Morph
	EOS    int
	Output []kansaiBoundState
}
type kansaiBoundReference struct {
	DLLHash, RulesHash, CaptureHash, Entry, Source string
	Cases                                          []kansaiBoundCase
}

func boundState(m *Morph) kansaiBoundState {
	return kansaiBoundState{m.Kind, m.Kind2, m.Up, m.Down, m.Moras, m.Annotation}
}

func TestKansaiBoundRulesOriginal(t *testing.T) {
	raw, err := os.ReadFile("testdata/kansai_bound_rules.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref kansaiBoundReference
	if err = json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.DLLHash != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || ref.Entry != "0x135e80" || len(ref.Cases) != 6 {
		t.Fatal("original reference provenance differs")
	}
	r, err := os.ReadFile("../../assets/rules/bound-rules-kansai.txt")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(r)) != ref.RulesHash {
		t.Fatal("Kansai rules differ from original reference")
	}
	f, err := os.Open("../../assets/rules/bound-rules-kansai.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rules, err := ParseBoundRules(f)
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, c := range ref.Cases {
		if len(c.Input) != len(c.Output) || c.EOS < 0 || c.EOS >= len(c.Input) {
			t.Fatal("invalid reference extent")
		}
		doc := &Document{Morphs: c.Input, EOS: c.Input[c.EOS]}
		doc.ApplyKansaiBoundRules(rules)
		for i, m := range doc.Morphs {
			if got := boundState(m); got != c.Output[i] {
				t.Fatalf("%q/%d %q got %+v want %+v", c.Text, i, m.Surface, got, c.Output[i])
			}
			total++
		}
	}
	if total != 40 {
		t.Fatal("incomplete reference")
	}
	t.Logf("6 original Kansai rule passes, %d morphs: boundary fields and reading cache identical", total)
}
