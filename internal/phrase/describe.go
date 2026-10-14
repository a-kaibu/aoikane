package phrase

import (
	"fmt"
	"strings"
)

// Describe renders the prosodic structure of a document in a stable text form
// used to compare the reconstruction with snapshots of the native engine.
func (d *Document) Describe() string {
	var b strings.Builder
	for si, s := range d.Sentences {
		fmt.Fprintf(&b, "S%d type=%d end=%d\n", si, s.Type, s.End)
		var end *PWord
		if s.Next != nil {
			end = s.Next.PWord
		}
		for p := s.PWord; p != nil && p != end; p = p.Next {
			var words []string
			stop := (*Morph)(nil)
			if p.Next != nil {
				stop = p.Next.First
			}
			for m := p.First; m != nil && m != stop; m = m.Next {
				words = append(words, m.Surface)
			}
			fmt.Fprintf(&b, "  P up=%d down=%d moras=%d strength=%.4f f38=%t [%s]\n",
				p.Up, p.Down, p.Moras, p.Strength, p.Flag38, strings.Join(words, "|"))
		}
	}
	for _, m := range d.Morphs {
		fmt.Fprintf(&b, "M %q kind=%d/%d\n", m.Surface, m.Kind, m.Kind2)
	}
	return b.String()
}
