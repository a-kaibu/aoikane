package frontend

import (
	"fmt"
	"strings"
)

// PhoneSequence joins the reconstructed mora, prosody, voicing, allophone and
// articulation passes. Acoustic features and recordings are never inputs.
type PhoneSequence struct {
	ProsodyTokens
	Phones       []Allophone `json:"phones"`
	Articulation []uint32    `json:"articulation"`
}

// CompilePhones consumes selected kana/accent data. Text reading and accent
// connection are upstream. The caller supplies the inspected voicing flags.
func (e *MoraEncoder) CompilePhones(s KanaSentence, voicingFlags uint32) (PhoneSequence, error) {
	prosody, err := e.CompileProsody(s)
	if err != nil {
		return PhoneSequence{}, err
	}
	out := PhoneSequence{ProsodyTokens: prosody}
	out.Phones = append(out.Phones, Allophone{ID: 0, Label: "--_S", Info: 0xcf00, BoundaryCode: 0x10})
	var kinds = []int{2}
	setLabel := func(index, at int, value byte) {
		b := []byte(out.Phones[index].Label)
		b[at] = value
		out.Phones[index].Label = string(b)
	}
	for ai, a := range s.Accents {
		end := len(s.Timeline) - 1
		if ai+1 < len(s.Accents) {
			end = s.Accents[ai+1].Begin
		}
		moraEnd := end
		if s.Timeline[end-1].Type == "pause" {
			moraEnd--
		}
		for ti := a.Begin; ti < moraEnd; ti++ {
			item := s.Timeline[ti]
			var indices []int
			if item.Mora == "ー" {
				previous := len(out.Phones) - 1
				for previous > 0 && out.Phones[previous].ID == 0 {
					previous--
				}
				id := out.Phones[previous].ID
				if id >= 1 && id <= 5 {
					out.Phones[previous].Info |= 0x2000
					setLabel(previous, 2, ':')
					indices = []int{id}
				} else if id == 50 || id == 51 {
					out.Phones[previous].Info |= 0x10
					setLabel(previous, 2, ':')
					indices = []int{id}
				}
			} else {
				units, err := e.EncodePronunciation(item.Mora, false)
				if err != nil || len(units) != 1 {
					return PhoneSequence{}, fmt.Errorf("unsupported mora %q", item.Mora)
				}
				indices = units[0].PhoneIndices
			}
			for _, index := range indices {
				def := e.Phones[index]
				out.Phones = append(out.Phones, Allophone{ID: def.ID, Label: strings.Repeat("_", 4), Info: def.Info & 0xfffe})
				out.Phones[len(out.Phones)-1].Label = def.Label + strings.Repeat("_", 4-len(def.Label))
				kinds = append(kinds, 0)
			}
			last := len(out.Phones) - 1
			mi := ti - a.Begin + 1
			if mi == a.Up {
				out.Phones[last].BoundaryCode = 8
				setLabel(last, 3, '^')
			}
			if mi == a.Down {
				out.Phones[last].BoundaryCode = 4
				setLabel(last, 3, '!')
			}
			if mi == a.Up && mi == a.Down {
				out.Phones[last].BoundaryCode = 12
				setLabel(last, 3, '*')
			}
		}
		if ai+1 < len(s.Accents) || moraEnd != end {
			kind := map[byte]uint8{'|': 2, '/': 3, '$': 1}[a.Bound[0]]
			last := len(out.Phones) - 1
			out.Phones[last].BoundaryCode |= kind
			setLabel(last, 3, a.Bound[0])
		}
		if moraEnd != end {
			pause := s.Timeline[moraEnd].Pause
			code, ok := map[string]uint8{"_0": 0x16, "_1": 0x1c, "_2": 0x14}[pause]
			if !ok {
				return PhoneSequence{}, fmt.Errorf("initial phone generation for pause %q is not reconstructed", pause)
			}
			label := map[string]string{"_0": "--_B", "_1": "--_P", "_2": "--_-"}[pause]
			out.Phones = append(out.Phones, Allophone{ID: 0, Label: label, Info: 0xcf00, BoundaryCode: code})
			kinds = append(kinds, 0)
		}
	}
	last := len(out.Phones) - 1
	if out.Phones[last].BoundaryCode&3 == 0 {
		out.Phones[last].BoundaryCode |= 1
		setLabel(last, 3, '$')
	}
	endCode := map[string]uint8{"F": 0x18, "N": 0x15, "A": 0x1d, "H": 0x1a, "R": 0x19}[s.EOS]
	endKind := map[string]int{"F": 3, "N": 4, "A": 7, "H": 6, "R": 5}[s.EOS]
	out.Phones = append(out.Phones, Allophone{ID: 0, Label: "--_" + s.EOS, Info: 0xcf00, BoundaryCode: endCode})
	kinds = append(kinds, endKind)
	if len(out.Phones) != len(prosody.Phoneme) {
		return PhoneSequence{}, fmt.Errorf("prosody/phone count differs")
	}
	context := make([]DevoicePhone, len(out.Phones))
	for i, p := range out.Phones {
		context[i] = DevoicePhone{Allophone: p, Accent: int(prosody.Accent[i]), PauseKind: kinds[i]}
	}
	if err := ApplyDevoicing(context, voicingFlags); err != nil {
		return PhoneSequence{}, err
	}
	for i, p := range context {
		out.Phones[i] = p.Allophone
	}
	// Articulation is formed before contextual consonant/nasal rewriting. The
	// native bitmap survives those later changes to Info and Label.
	out.Articulation, err = e.ModelArticulation(out.Phones)
	if err != nil {
		return PhoneSequence{}, err
	}
	if err := ApplyAllophones(out.Phones); err != nil {
		return PhoneSequence{}, err
	}
	// RVA 0x1f330 follows the allophone pass. Fricative-like geminates use
	// the following phone ID and its stored articulation with bit 26 set.
	// Stops, affricates, vowels and the moraic nasal retain the closure ID.
	for i := 0; i+1 < len(out.Phones); i++ {
		next := out.Phones[i+1].ID
		if out.Phones[i].ID != 51 || next <= 5 || next == 50 ||
			(next >= 6 && next <= 11) || (next >= 18 && next <= 25) ||
			(next >= 33 && next <= 38) {
			continue
		}
		out.Phones[i].ID = next
		out.Articulation[i] = out.Articulation[i+1] | 1<<26
	}
	return out, nil
}
