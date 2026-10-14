package phrase

import (
	"fmt"

	"github.com/a-kaibu/aoikane/internal/frontend"
)

// accentRecordCount is 159330: the number of accent entries in an accent text
// (a leading digit and every comma followed by a digit).
func accentRecordCount(text string) int {
	n := 0
	if len(text) > 0 && digit(text[0]) {
		n = 1
	}
	for i := 0; i < len(text); i++ {
		if text[i] == ',' && digit(at(text, i+1)) {
			n++
		}
	}
	return n
}

// updateReading is 177380 for a morph whose pronunciation and accent are
// final: it recomputes the cached rise, fall and mora count.
func updateReading(m *Morph) error {
	if c := m.SelectedCandidate(); c != nil {
		rec, err := frontend.ParseAccentText(c.Accent, 0)
		if err != nil {
			return err
		}
		m.Up, m.Down = rec.First, rec.Second
	}
	moras := int32(0)
	if c := m.SelectedCandidate(); c != nil {
		for _, r := range c.Pronunciation {
			if startsMora(r) {
				moras++
			}
		}
	}
	m.Moras = moras
	if m.Down > m.Moras {
		m.Down = m.Moras
	}
	if m.Down != 0 {
		if m.Up > m.Down {
			m.Up = m.Down
		} else if m.Up == 0 {
			m.Up = 1
		}
	}
	return nil
}

// formatAccent is 159a50: the accent text of a word part. A negative speaker
// omits the ",s<n>" marker.
func formatAccent(up, down, moras, speaker int32) string {
	cap99 := int32(99)
	if up < 0 {
		switch {
		case moras != 1 && down != 1:
			up = 2
		case moras != 1:
			up = 1
			goto moraCheck
		case down != 1:
			up = 0
		default:
			up = 1
			if speaker < 100 {
				cap99 = speaker
			}
			goto format
		}
	} else if up > 99 {
		up = 99
	}
	if down < 0 {
		down = 0
	} else if down > 99 {
		down = 99
	}
moraCheck:
	if moras < 0 {
		return "*"
	}
	if moras < 100 {
		if speaker < 100 {
			cap99 = speaker
		}
		if moras == 0 {
			return "*"
		}
	} else {
		moras = cap99
		if speaker < 100 {
			cap99 = speaker
		}
	}
format:
	if cap99 < 0 {
		return fmt.Sprintf("%d-%d-%d", up, down, moras)
	}
	return fmt.Sprintf("%d-%d-%d,s%d", up, down, moras, cap99)
}

// speakerKind maps a speaker marker to the boundary kind it implies.
func speakerKind(speaker int32) (int32, bool) {
	switch speaker {
	case 0:
		return 0x15, true
	case 1:
		return 0xe, true
	case 2:
		return 0xd, true
	case 3:
		return 0x10, true
	}
	return 0, false
}

// splitByAccent is the splitting branch of 191710: a word whose accent text has
// several entries becomes one morph per entry, each carrying its share of the
// pronunciation. The first part keeps the original node.
func (d *Document) splitByAccent(m *Morph, count int) (*Morph, error) {
	c := m.SelectedCandidate()
	rs := []rune(m.Pronunciation())
	type part struct {
		first, second, third, speaker int32
		pron                          string
	}
	parts := make([]part, 0, count)
	start, offset := 0, 0
	for k := 0; k < count; k++ {
		rec, err := frontend.ParseAccentText(c.Accent, start)
		if err != nil {
			return nil, err
		}
		pron := moraRange(m, offset, int(rec.Third))
		parts = append(parts, part{rec.First, rec.Second, rec.Third, rec.Speaker, pron})
		offset += int(rec.Third)
		if k < count-1 {
			if rec.Next <= start {
				return nil, fmt.Errorf("accent text %q ends before entry %d", c.Accent, k+2)
			}
			start = rec.Next
		}
	}
	_ = rs
	original := *m
	connection := c.Connection
	prevSpeaker := int32(-1)
	prev := m
	next := m.Next
	for k, p := range parts {
		node := m
		if k > 0 {
			node = &Morph{
				POS: original.POS, POS2: original.POS2, POS3: original.POS3,
				Kind2: original.Kind2, F190: original.F190, F194: original.F194,
				TagPtr: original.TagPtr, TagLen: original.TagLen,
			}
			node.Prev, prev.Next = prev, node
			prev = node
		}
		conn := "*"
		if k == 0 {
			conn = connection
		} else if prevSpeaker == 3 {
			conn = "S3"
		}
		moras := int32(0)
		for _, r := range p.pron {
			if startsMora(r) {
				moras++
			}
		}
		node.Cands = []Candidate{{Reading: p.pron, Pronunciation: p.pron, Accent: formatAccent(p.first, p.second, moras, p.speaker), Connection: conn}}
		node.Selected = 0
		kind := int32(0)
		if k < count-1 {
			kind = 0x15
			if v, ok := speakerKind(p.speaker); ok {
				kind = v
			}
		} else {
			kind = original.Kind
			if original.Kind == 0 || original.Kind == 0x15 {
				if v, ok := speakerKind(p.speaker); ok {
					kind = v
				}
			}
		}
		node.Kind = kind
		prevSpeaker = p.speaker
	}
	prev.Next = next
	if next != nil {
		next.Prev = prev
	}
	d.renumber()
	return prev, nil
}

func (d *Document) renumber() {
	d.Morphs = d.Morphs[:0]
	var head *Morph
	for m := d.EOS; m != nil; m = m.Prev {
		head = m
	}
	for m := head; m != nil; m = m.Next {
		m.Index = len(d.Morphs)
		d.Morphs = append(d.Morphs, m)
	}
}

// finalPass is 1902a0 without the detached-chain expansion: accent records
// that name a speaker mark the boundary kind, words with several accent
// entries are split, then every reading cache is refreshed.
func (d *Document) finalPass() error {
	if err := d.expandChains(); err != nil {
		return err
	}
	for m := d.firstMorph(); m != nil && m != d.EOS; m = m.Next {
		c := m.SelectedCandidate()
		if c == nil {
			continue
		}
		count := accentRecordCount(c.Accent)
		switch {
		case count >= 2:
			last, err := d.splitByAccent(m, count)
			if err != nil {
				return err
			}
			m = last
		case count == 1:
			if m.Kind != 0 && m.Kind != 0x15 {
				continue
			}
			rec, err := frontend.ParseAccentText(c.Accent, 0)
			if err != nil {
				return err
			}
			if v, ok := speakerKind(rec.Speaker); ok {
				m.Kind = v
			}
		}
	}
	for m := d.firstMorph(); m != nil && m != d.EOS; m = m.Next {
		if err := updateReading(m); err != nil {
			return err
		}
	}
	return nil
}

func (d *Document) firstMorph() *Morph {
	var head *Morph
	for m := d.EOS; m != nil; m = m.Prev {
		head = m
	}
	return head
}

// expandChains is the first loop of 1902a0 (1903a0): a detached word chain is
// spliced in after its owner, whose surface becomes the concatenation of all
// surfaces; the chain's words take kind 21 except the last, which inherits the
// owner's kind and label.
func (d *Document) expandChains() error {
	for m := d.firstMorph(); m != nil && m != d.EOS; m = m.Next {
		if m.ExpFirst == nil {
			continue
		}
		if m.ExpLast == nil || m.Next == nil {
			return fmt.Errorf("expansion of %q lacks a last word or a following boundary", m.Surface)
		}
		chain := []*Morph{m}
		seen := map[*Morph]bool{m: true, m.Next: true}
		for n := m.ExpFirst; ; n = n.Next {
			if n == nil || seen[n] || len(chain) >= 65536 {
				return fmt.Errorf("invalid detached chain of %q", m.Surface)
			}
			seen[n] = true
			chain = append(chain, n)
			if n == m.ExpLast {
				break
			}
		}
		surface := ""
		for _, n := range chain {
			surface += n.Surface
		}
		last, following := m.ExpLast, m.Next
		kind, annotation := m.Kind, m.Annotation
		m.Next = m.ExpFirst
		m.Next.Prev = m
		last.Next = following
		following.Prev = last
		m.ExpFirst, m.ExpLast = nil, nil
		for _, n := range chain {
			n.Kind = 21
			n.Surface = ""
		}
		last.Kind, last.Annotation = kind, annotation
		m.Surface = surface
		d.renumber()
		m = last
	}
	return nil
}
