package phrase

import (
	"github.com/a-kaibu/aoikane/internal/frontend"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestKansaiChunkBoundOriginal(t *testing.T) {
	data, err := os.ReadFile("testdata/kansai_chunk_bound.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		DLLHash, Entry, RulesHash string
		Cases                     []struct {
			kansaiBoundCase
			Chunks []struct {
				First  int
				Fields [4]int32
			}
		}
	}
	if err = json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.DLLHash != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || ref.Entry != "0x137520" || len(ref.Cases) != 6 {
		t.Fatal("original Kansai reference differs")
	}
	r, err := os.ReadFile("../../assets/rules/kansai-chunk-bound.json")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(r)) != ref.RulesHash {
		t.Fatal("original rules differ")
	}
	rules, err := ParseKansaiChunkRules(strings.NewReader(string(r)))
	if err != nil {
		t.Fatal(err)
	}
	codec, err := frontend.OpenCP932Codec("../../assets/data/cp932.json")
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, c := range ref.Cases {
		if len(c.Input) != len(c.Output) || c.EOS < 0 || c.EOS >= len(c.Input) || len(c.Chunks) < 2 {
			t.Fatal("invalid reference extent")
		}
		doc := &Document{Morphs: c.Input, EOS: c.Input[c.EOS]}
		for i, m := range doc.Morphs {
			if i > 0 {
				m.Prev = doc.Morphs[i-1]
			}
			if i+1 < len(doc.Morphs) {
				m.Next = doc.Morphs[i+1]
			}
		}
		var previous *Chunk
		for _, r := range c.Chunks {
			if r.First < 0 || r.First >= len(doc.Morphs) {
				t.Fatal("invalid reference chunk")
			}
			chunk := &Chunk{First: doc.Morphs[r.First], F28: r.Fields[0], F2c: r.Fields[1], F30: r.Fields[2], F34: r.Fields[3], Prev: previous}
			if previous == nil {
				doc.Sentences = []*Sentence{{Chunk: chunk}}
			} else {
				previous.Next = chunk
			}
			previous = chunk
		}
		if err = doc.ApplyKansaiChunkBound(rules, codec); err != nil {
			t.Fatal(err)
		}
		for i, m := range doc.Morphs {
			if got := boundState(m); got != c.Output[i] {
				t.Fatalf("%q/%q got %+v want %+v", c.Text, m.Surface, got, c.Output[i])
			}
			total++
		}
	}
	if total != 40 {
		t.Fatal("incomplete reference")
	}
	t.Logf("6 texts %d original morphs exact", total)
}
