package phrase

import "strings"

var startTags = map[int32]string{0: "<S>", 1: "<D>"}

var endTags = [...]string{"<F>", "<N>", "<R>", "<W>", "<H>", "<A>", "<J>", "<M>", "<C>", "<T>"}

// endOfKind2 maps the sentence-end styles 0x1f..0x28 of the last audible
// morph to the end marker (the switch in 1aa4f0).
var endOfKind2 = map[int32]int32{0x1f: 0, 0x20: 2, 0x21: 3, 0x22: 4, 0x23: 5, 0x24: 6, 0x25: 7, 0x26: 1, 0x27: 8, 0x28: 9}

// pwordText is the audible text of a prosodic word with the rise marker "^"
// before mora Up and the fall marker "!" after mora Down.
func pwordText(p *PWord, kansai bool) string {
	var units []string
	end := p.endMorph()
	for m := p.First; m != nil && m != end; m = m.Next {
		if m.Moras <= 0 || m.TagPtr != 0 {
			continue
		}
		for _, r := range m.Pronunciation() {
			if startsMora(r) || len(units) == 0 {
				units = append(units, string(r))
			} else {
				units[len(units)-1] += string(r)
			}
		}
	}
	// The native post-pass drops a trailing "!" (fall after the last mora).
	// Standard Japanese also drops the rise of a single-mora falling word;
	// Kansai retains it (16d400 receives Mode == 1 as its fifth argument).
	rise, fall := int(p.Up), int(p.Down)
	if fall > 0 && fall >= len(units) {
		fall = 0
		if !kansai && rise == 1 && len(units) == 1 {
			rise = 0
		}
	}
	var b strings.Builder
	for i := 0; i <= len(units); i++ {
		if rise > 0 && rise-1 == i {
			b.WriteByte('^')
		}
		if i < len(units) {
			b.WriteString(units[i])
		}
		if fall > 0 && fall-1 == i {
			b.WriteByte('!')
		}
	}
	return b.String()
}

// pwordSuffix is 16cee0: the boundary marker that follows a prosodic word.
func pwordSuffix(p *PWord) string {
	if p.Next == nil {
		return ""
	}
	if p.Next.Owner != nil {
		return ""
	}
	if p.Next.First != nil && p.First != nil {
		stop := p.First.Prev
		for m := p.Next.First.Prev; m != nil && m != stop; m = m.Prev {
			if m.Kind != 0 {
				switch m.Kind {
				case 2, 4, 8, 9, 0xd, 0x16, 0x1d:
					return "$2_2"
				case 3, 7, 0xe, 0x1c:
					return "$1_1"
				case 0xf:
					return "$2_1"
				case 0x10:
					if p.Down < 1 || p.Moras <= p.Down {
						return "/1"
					}
					return "/0"
				case 0x18:
					return "|0"
				}
				break
			}
			if m.Moras > 0 || m.TagPtr != 0 {
				break
			}
		}
	}
	if p.Next == nil || p.Next.Next == nil || p.Next.Owner != nil {
		return ""
	}
	return "|0"
}

// Format renders the sentence in the engine's talk notation, for example
// "<S>ト^ーキョーノ|0^テ!ンキワ|0ハ^レ!デス<F>".
func (d *Document) Format(s *Sentence) string {
	var b strings.Builder
	b.WriteString(startTags[s.Type])
	end := s.endPWord()
	for p := s.PWord; p != nil && p != end; p = p.Next {
		last := p.Next == nil || p.Next.Next == nil || p.Next.Owner != nil
		if last && s.End != 2 && s.End != 8 {
			if v := p.kind2Of(); v != 0 {
				if e, ok := endOfKind2[v]; ok {
					s.End = e
				}
			}
		}
		b.WriteString(pwordText(p, d.Mode == 1))
		b.WriteString(pwordSuffix(p))
	}
	if s.End >= 0 && int(s.End) < len(endTags) {
		b.WriteString(endTags[s.End])
	}
	return b.String()
}
