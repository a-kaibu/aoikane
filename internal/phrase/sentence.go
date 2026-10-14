package phrase

import (
	"fmt"
	"strings"
)

// setPronunciation is 16baf0 with no accent override: it replaces the selected
// candidate's pronunciation and recounts the reading's moras.
func setPronunciation(m *Morph, pron string) error {
	c := m.SelectedCandidate()
	if c == nil {
		return nil
	}
	c.Pronunciation = pron
	first, second := 0, 0
	if len(c.Accent) > 0 {
		first, second = parseLeadingAccent(c.Accent)
	}
	moras := int32(0)
	for _, r := range pron {
		if startsMora(r) {
			moras++
		}
	}
	up, down := int32(first), int32(second)
	if up < 0 || moras < up {
		up = 0
	}
	if down < 0 || moras < down {
		down = 0
	}
	m.Up, m.Down, m.Moras = up, down, moras
	return nil
}

// parseLeadingAccent returns the first two numbers of an accent text as read
// by 1590b0 (an optional "p,"/"s<n>," prefix is skipped).
func parseLeadingAccent(text string) (first, second int) {
	if strings.HasPrefix(text, "*") {
		return 2, 0
	}
	return parseAccentPair(text)
}

// a9ea0 is 1a9ea0: pronunciation corrections before the phrases are built.
func (d *Document) a9ea0(s *Sentence) error {
	if s.Chunk == nil {
		return nil
	}
	end := s.endMorph()
	for m := s.Chunk.First; m != nil; m = m.Next {
		if m == end {
			return nil
		}
		pron := m.Pronunciation()
		if m.POS == 0xd030000 {
			if pron == "" && utf8Len(m.Surface) == 1 {
				r := []rune(m.Surface)[0]
				switch {
				case r == 'ヶ':
					if err := setPronunciation(m, "ガ"); err != nil {
						return err
					}
				case isHiragana(r):
					if err := setPronunciation(m, string(hiraganaToKatakana(r))); err != nil {
						return err
					}
				}
			}
		} else if m.POS == 0x3000000 && pron == "イウ" {
			if err := setPronunciation(m, "ユー"); err != nil {
				return err
			}
		}
		rs := []rune(m.Pronunciation())
		changed := false
		for i := 0; i+1 < len(rs); i++ {
			if (rs[i] == 'ク' || rs[i] == 'グ') && rs[i+1] == 'ヮ' {
				rs[i+1] = 'ァ'
				changed = true
			}
		}
		if changed {
			if err := setPronunciation(m, string(rs)); err != nil {
				return err
			}
		}
	}
	return nil
}

func utf8Len(s string) int { return len([]rune(s)) }

func isHiragana(r rune) bool { return r >= 'ぁ' && r <= 'ん' || r == 'ー' }

func hiraganaToKatakana(r rune) rune {
	if r >= 'ぁ' && r <= 'ん' {
		return r + 0x60
	}
	return r
}

// endKind computes the sentence end class at the start of the 1aa4f0 loop.
func (d *Document) sentenceEnds(s *Sentence) {
	if s.Type == 2 {
		if s.Chunk == nil || s.Chunk.First == nil || s.Chunk.First.F190 == 2 {
			s.Type = 0
		} else {
			s.Type = s.Chunk.First.F190
		}
	}
	if s.End != 10 {
		return
	}
	last := s.endMorph()
	if last == nil || last.Prev == nil {
		s.End = 0
		return
	}
	last = last.Prev
	if last.F194 != 10 {
		s.End = last.F194
		return
	}
	switch last.Base {
	case "、":
		s.End = 8
	case "？":
		s.End = 2
	case "。", "！":
		s.End = 0
	default:
		s.End = 0
		for m := last; m != nil; m = m.Prev {
			if m.Pronunciation() != "" {
				if m.POS&0xff000000 == 0x1000000 {
					s.End = 1
				}
				break
			}
		}
	}
}

// BuildSentence runs the per-sentence part of 1aa4f0 up to the phrase limits.
func (d *Document) BuildSentence(s *Sentence) error {
	if err := d.a9ea0(s); err != nil {
		return err
	}
	d.sentenceEnds(s)
	if d.aff70(s) {
		d.af570(s)
	}
	return nil
}

// Build processes every sentence of the document.
func (d *Document) Build() error {
	if err := d.finalPass(); err != nil {
		return err
	}
	for _, s := range d.Sentences {
		if err := d.BuildSentence(s); err != nil {
			return fmt.Errorf("sentence: %w", err)
		}
	}
	return nil
}
