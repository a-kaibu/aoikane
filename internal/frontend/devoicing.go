package frontend

import "fmt"

// DevoicePhone supplies the contextual fields read by native RVA 0x21210.
// Labels may contain D/V overrides from the earlier pronunciation stage.
type DevoicePhone struct {
	Allophone
	Accent int `json:"accent"`
	// PauseKind is native offset 0x60, not the model's punc category.
	PauseKind int `json:"pause_kind"`
}

// ApplyDevoicing reproduces the contextual vowel voicing pass. Flags is the
// first configuration word (the inspected voice uses 0xef). This expects the
// complete sentence padding and already generated accent/boundary information.
// Articulation must be computed after this operation.
func ApplyDevoicing(phones []DevoicePhone, flags uint32) error {
	if len(phones) < 2 || len(phones) > 8192 {
		return fmt.Errorf("invalid devoicing phone count")
	}
	for _, p := range phones {
		if p.ID < 0 || p.ID >= 52 || len(p.Label) != 4 || p.Accent < 0 || p.Accent > 3 || p.PauseKind < 0 || p.PauseKind > 11 {
			return fmt.Errorf("invalid devoicing phone")
		}
		for i := range p.Label {
			if p.Label[i] > 127 {
				return fmt.Errorf("non-ASCII devoicing label")
			}
		}
	}
	change := func(i int, unvoiced bool) {
		p := &phones[i]
		if p.Label[1] == 'D' || p.Label[1] == 'V' {
			return
		}
		label := []byte(p.Label)
		if unvoiced {
			p.Info |= 0x4000
			label[2] = '0'
		} else {
			p.Info &^= 0x4000
			label[2] = '_'
		}
		p.Label = string(label)
	}
	voicelessConsonant := func(info uint16) bool { return info&0x8100 == 0x100 }
	unvoicedVowel := func(info uint16) bool { return info&0xc000 == 0xc000 }
	for i := 1; i+1 < len(phones); i++ {
		p := &phones[i]
		if flags&1 != 0 && voicelessConsonant(phones[i-1].Info) && voicelessConsonant(phones[i+1].Info) && phones[i+1].Info&0xc0 != 0x40 && (p.ID == 2 || p.ID == 3) {
			if flags&0x10 == 0 || p.BoundaryCode&0x10 != 0 || p.BoundaryCode&4 == 0 {
				change(i, true)
			}
		}
		if flags&2 != 0 && voicelessConsonant(phones[i-1].Info) && p.Accent == 1 && (flags&0x80 == 0 || phones[i+1].ID == 0) && (p.ID == 2 || p.ID == 3) {
			change(i, true)
		}
		if unvoicedVowel(p.Info) {
			if i > 2 && phones[i-2].Info&0x80c0 == 0x40 {
				change(i, false)
			}
			if phones[i+1].Info&0x80c0 == 0x40 {
				change(i, false)
			}
		}
	}
	if flags&0x60 != 0 {
		for i := 3; i < len(phones); i++ {
			if !unvoicedVowel(phones[i].Info) || phones[i].Info&0xf00 == 0xf00 || !unvoicedVowel(phones[i-2].Info) || phones[i-2].Info&0xf00 == 0xf00 {
				continue
			}
			if flags&0x40 != 0 && i < len(phones)-2 && unvoicedVowel(phones[i+2].Info) {
				change(i, false)
			} else if flags&0x20 != 0 {
				change(i-2, false)
			}
		}
	}
	if flags&0x400 != 0 {
		for i := 3; i+1 < len(phones); i++ {
			p := &phones[i]
			if p.Accent != 1 || phones[i-1].ID != 12 || p.ID != 3 || phones[i+1].ID != 0 {
				continue
			}
			if flags&0x800 != 0 && phones[i+1].PauseKind == 0 {
				continue
			}
			if (phones[i-2].ID == 4 && phones[i-3].ID == 23) || (phones[i-2].ID == 1 && phones[i-3].ID == 42) {
				change(i, true)
			}
		}
	}
	last := len(phones) - 1
	if len(phones) > 2 && unvoicedVowel(phones[last-1].Info) && phones[last].PauseKind <= 9 && (uint32(0x362)&(1<<uint(phones[last].PauseKind))) != 0 {
		change(last-1, false)
	}
	return nil
}
