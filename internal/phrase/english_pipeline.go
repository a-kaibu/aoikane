package phrase

import (
	"fmt"
	"strings"

	"github.com/a-kaibu/aoikane/internal/frontend"
)

// groupWords applies native kana/English merging and cleanup before CaboCha. The
// merged words retain their original text spans, while known words retain
// their dictionary entries and all pronunciation candidates.
func (f *Frontend) groupWords(words []frontend.Word) ([]frontend.Word, map[int]*Morph, error) {
	if f.English == nil {
		return words, nil, nil
	}
	input := make([]frontend.KanaGroupWord, len(words))
	for i, w := range words {
		mw := frontend.KanaMergeWord{Surface: w.Text, Base: w.Text, POS: 0xd030000}
		if !w.Entry.Unknown {
			lw, err := f.Morphology.LanguageWord(w)
			if err != nil {
				return nil, nil, err
			}
			mw.Base, mw.POS = lw.Base, lw.POS
			cands, err := frontend.DictionaryCandidates(w.Entry)
			if err != nil {
				return nil, nil, err
			}
			if len(cands) > 0 {
				mw.Pronunciation = cands[0].Pronunciation
			}
		}
		input[i].KanaMergeWord = mw
	}
	merged, err := frontend.MergeKanaEnglishGroup(input, f.Codec, f.UnknownKana, f.UnknownAccent, f.English.Compile)
	if err != nil {
		return nil, nil, err
	}
	out := make([]frontend.Word, 0, len(merged))
	generated := make(map[int]*Morph)
	position := 0
	for _, mw := range merged {
		begin := position
		var surface strings.Builder
		for position < len(words) && surface.Len() < len(mw.Surface) {
			surface.WriteString(words[position].Text)
			position++
		}
		if position == begin || surface.String() != mw.Surface {
			return nil, nil, fmt.Errorf("word merge lost source span for %q", mw.Surface)
		}
		w := words[begin]
		if mw.Generated != nil {
			var form, kind uint32
			if !w.Entry.Unknown {
				lw, err := f.Morphology.LanguageWord(w)
				if err != nil {
					return nil, nil, err
				}
				form, kind = lw.Form, lw.Type
			}
			w.Text, w.End = mw.Surface, words[position-1].End
			w.Entry = frontend.Entry{Unknown: true, Surface: mw.Surface}
			r := mw.Generated
			up := 1
			if r.Accent.Up != nil {
				up = *r.Accent.Up
			} else if r.Accent.Down != 1 && r.Accent.Moras != 1 {
				up = 2
			}
			generated[len(out)] = &Morph{Surface: mw.Surface, Base: mw.Base, POS: mw.POS, POS2: form, POS3: kind, Moras: -1, Up: -1, Down: -1,
				Cands: []Candidate{{Reading: r.Pronunciation, Pronunciation: r.Pronunciation,
					Accent: fmt.Sprintf("%d-%d-%d", up, r.Accent.Down, r.Accent.Moras), Connection: "*"}}}
		} else if position != begin+1 {
			return nil, nil, fmt.Errorf("word merge changed a dictionary word without a generated reading")
		}
		out = append(out, w)
	}
	return f.cleanupGroupedWords(out, generated)
}

// The native cleanup follows kana/English merging, before CaboCha and reading
// selection. Keep the dictionary alternatives when it only changes POS.
func (f *Frontend) cleanupGroupedWords(words []frontend.Word, generated map[int]*Morph) ([]frontend.Word, map[int]*Morph, error) {
	morphs := make([]*Morph, len(words))
	for i, w := range words {
		m := generated[i]
		if m == nil {
			if w.Entry.Unknown {
				m = &Morph{Surface: w.Text, Base: w.Text, POS: 0xd030000}
			} else {
				lw, err := f.Morphology.LanguageWord(w)
				if err != nil {
					return nil, nil, err
				}
				cs, err := frontend.DictionaryCandidates(w.Entry)
				if err != nil {
					return nil, nil, err
				}
				m = &Morph{Surface: w.Text, Base: lw.Base, POS: lw.POS, POS2: lw.Form, POS3: lw.Type, Moras: -1, Up: -1, Down: -1}
				for _, c := range cs {
					if c.Reading == "" {
						c.Reading = "＊"
					}
					m.Cands = append(m.Cands, Candidate{c.Reading, c.Pronunciation, c.Accent, c.Connection})
				}
			}
		}
		morphs[i] = m
	}
	word := func(m *Morph) frontend.CleanupWord {
		return frontend.CleanupWord{KanaMergeWord: frontend.KanaMergeWord{Surface: m.Surface, Base: m.Base, POS: m.POS, Pronunciation: m.Pronunciation()}}
	}
	out := make([]frontend.Word, 0, len(words))
	fixed := map[int]*Morph{}
	left := frontend.CleanupWord{HasChildren: true}
	for i, m := range morphs {
		right := frontend.CleanupWord{HasChildren: true}
		if i+1 < len(morphs) {
			right = word(morphs[i+1])
		}
		var moras *int
		if m.Surface == "・" && left.POS == 0x01010102 && right.POS == 0x01010101 && i+1 < len(morphs) && morphs[i+1].SelectedCandidate() != nil {
			c := morphs[i+1].SelectedCandidate()
			r, err := frontend.ParseAccentText(c.Accent, 0)
			if err != nil {
				return nil, nil, err
			}
			n := int(r.Second)
			moras = &n
		}
		result, err := frontend.CleanupWordNeighbors(left, word(m), right, moras)
		if err != nil {
			return nil, nil, err
		}
		if result.ResetReading {
			m.Cands = []Candidate{{Reading: "｜", Accent: "*", Connection: "*"}}
			generated[i] = m
		}
		if i+1 < len(morphs) && result.Next.POS != morphs[i+1].POS {
			morphs[i+1].POS = result.Next.POS
			generated[i+1] = morphs[i+1]
		}
		if result.Keep {
			if generated[i] != nil {
				fixed[len(out)] = m
			}
			out = append(out, words[i])
			left = word(m)
		}
	}
	return out, fixed, nil
}
