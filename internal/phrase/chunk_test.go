package phrase

import (
	"path/filepath"
	"testing"

	"github.com/a-kaibu/aoikane/internal/cabocha"
)

// TestChunker compares the CRF chunker's bunsetsu boundaries with the engine's.
func TestChunker(t *testing.T) {
	codec := testCodec(t)
	model, err := cabocha.LoadCRF(filepath.Join("..", "..", "private", "cabocha", "standard-chunk.ipa.model"))
	if err != nil {
		t.Skip(err)
	}
	total, bad, tokens, wrong := 0, 0, 0, 0
	for _, c := range aoiCases(t, "stages_aoi", "stages_aoi3", "stages_aoi4") {
		d, err := c.Enter.Load(codec)
		if err != nil || len(d.Sentences) == 0 {
			continue
		}
		var toks [][]string
		var morphs []*Morph
		for _, m := range d.Morphs {
			if m == d.EOS {
				break
			}
			if m.POS&0xffff0000 == 0xb080000 {
				continue
			}
			s, err := codec.Encode(m.Surface)
			if err != nil {
				s = []byte(m.Surface)
			}
			pos, err := codec.Encode(cabocha.POSNames[m.POS])
			if err != nil {
				t.Fatal(err)
			}
			toks = append(toks, []string{string(s), string(pos)})
			morphs = append(morphs, m)
		}
		want := map[*Morph]bool{}
		for ch := d.Sentences[0].Chunk; ch != nil && ch.First != nil && ch.First != d.EOS; ch = ch.Next {
			want[ch.First] = true
		}
		tags := model.Tag(toks)
		total++
		mismatch := false
		for i, m := range morphs {
			got := tags[i] == "B"
			tokens++
			if got != want[m] {
				wrong++
				mismatch = true
			}
		}
		if mismatch {
			bad++
			if bad <= 12 {
				line := ""
				for i, m := range morphs {
					mark := ""
					if tags[i] == "B" {
						mark += "|"
					}
					if want[m] {
						mark += "*"
					}
					line += mark + m.Surface + " "
				}
				t.Logf("%s\n   %s", c.Text, line)
			}
		}
	}
	t.Logf("%d sentences, %d with differing chunk boundaries, %d of %d tokens", total, bad, wrong, tokens)
}
