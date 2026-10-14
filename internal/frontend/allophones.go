package frontend

import "fmt"

// Allophone holds the fields used by the original contextual consonant pass.
// Label has four ASCII bytes. Info is the word at native offset 0xd4, excluding
// the separately stored mora-head flag. BoundaryCode is the native prosody byte; upstream
// generation of that byte, accent and devoicing are separate pipeline stages.
type Allophone struct {
	ID           int    `json:"id"`
	Label        string `json:"label"`
	Info         uint16 `json:"info"`
	BoundaryCode uint8  `json:"boundary_code"`
}

func phraseEdge(code uint8) bool   { return code&0x10 == 0 && code&3 != 0 }
func voicedVowel(info uint16) bool { return info&0xc000 == 0x8000 }

// ApplyAllophones reproduces the inspected consonant pass (RVA 0x20c90).
// The first and last records are sentence padding and stay unchanged. This
// updates labels and phonetic information only, as the original pass does;
// it does not claim to produce complete acoustic model tokens.
func ApplyAllophones(phones []Allophone) error {
	for _, p := range phones {
		if p.ID < 0 || p.ID >= 52 || len(p.Label) != 4 {
			return fmt.Errorf("invalid allophone record")
		}
		for i := range p.Label {
			if p.Label[i] > 127 {
				return fmt.Errorf("allophone label must be ASCII")
			}
		}
	}
	for i := 1; i+1 < len(phones); i++ {
		p := &phones[i]
		previous, next := &phones[i-1], &phones[i+1]
		label := []byte(p.Label)
		switch p.ID {
		case 29: // Intervocalic h.
			if voicedVowel(previous.Info) && voicedVowel(next.Info) {
				p.Info &^= 0x100
				label[2] = 'v'
			}
		case 15: // Contextual z manner.
			p.Info &= 0xf0ff
			if voicedVowel(previous.Info) && voicedVowel(next.Info) {
				p.Info |= 0x200
				label[2] = 'f'
			} else {
				p.Info |= 0x600
			}
		case 9, 10, 11: // Nasalized g and explicit pronunciation overrides.
			switch label[2] {
			case 'G':
				label[2] = '_'
			case 'N':
				p.Info |= 0x20
				label[2] = 'n'
			default:
				if !phraseEdge(previous.BoundaryCode) && previous.Info&0x8000 != 0 && previous.Info&0xf00 != 0xf00 {
					p.Info |= 0x20
					label[2] = 'n'
				}
			}
		case 51: // Closure borrows the following consonant's place.
			if label[0] != next.Label[0] {
				label[1], label[2] = next.Label[0], next.Label[1]
			} else {
				label[1], label[2] = next.Label[1], next.Label[2]
			}
			if next.Info&0x8000 != 0 {
				p.Info = next.Info&0xff80 | 0x4080
			} else {
				p.Info = p.Info&0xff | next.Info&0xff40 | 0x40
			}
		case 50: // Moraic nasal assimilates across a non-boundary neighbor.
			if phraseEdge(p.BoundaryCode) || next.ID == 0 {
				label[1] = 'N'
				break
			}
			if next.Info&0x8000 != 0 {
				minor := p.Info & 0xf
				if p.Info&0x10 != 0 {
					minor |= 0x2000
				}
				p.Info = minor | next.Info&0x9f00 | 0x1000 | p.Info&0x40
				label[1] = next.Label[0]
				if label[0] == next.Label[0] {
					label[1] = next.Label[1]
				}
				break
			}
			switch next.Info & 0x7000 {
			case 0, 0x1000:
				p.Info = p.Info&0xff | 0x1400
				label[1] = 'g'
			case 0x2000, 0x3000:
				p.Info = p.Info&0xff | 0x3400
				label[1] = 'j'
			case 0x5000:
				p.Info = p.Info&0xdfff | 0x5000
				label[1] = 'm'
			case 0x4000:
				if next.Info&0xe00 == 0x200 {
					p.Info = 0x9601
					label[1] = 'u'
				} else {
					p.Info = p.Info&0xff | 0x4400
					label[1] = 'n'
				}
			}
		}
		p.Label = string(label)
	}
	return nil
}
