package frontend

import "fmt"

// ProsodyTokens contains the token dictionary fields that depend
// on mora/phrase position, rather than contextual articulation. Each mora's
// values are repeated for all of its phones. Padding and pauses have zero
// accent/position features. Articulation and synthesis controls are separate.
type ProsodyTokens struct {
	Phoneme        []int64   `json:"phn"`
	Accent         []int64   `json:"acc"`
	Punctuation    []int64   `json:"punc"`
	MoraPosition   []float32 `json:"nima"`
	PhrasePosition []float32 `json:"niab"`
	Boundaries     []int     `json:"boundaries"`
}

// CompileProsody expands an already selected accent-bearing reading. It does
// not perform text analysis, devoicing, contextual phonetics or articulation.
// The state transitions follow 0x1ecd3..0x1ed08: a rise sets state 3, except after
// a '/' boundary, where it sets 2; the mora after a fall returns to state 1.
func (e *MoraEncoder) CompileProsody(s KanaSentence) (ProsodyTokens, error) {
	var out ProsodyTokens
	if s.BOS != "S" || len(s.Accents) == 0 || len(s.Timeline) < 2 || len(s.Timeline) > 8190 {
		return out, fmt.Errorf("unsupported or empty kana sentence")
	}
	endPunc := map[string]int64{"F": 2, "N": 2, "A": 3, "H": 4, "R": 5}
	if _, ok := endPunc[s.EOS]; !ok {
		return out, fmt.Errorf("unsupported kana sentence end %q", s.EOS)
	}
	last := s.Timeline[len(s.Timeline)-1]
	if last.Type != "pause" || last.Pause != "eos" {
		return out, fmt.Errorf("missing kana sentence end pause")
	}
	appendPhone := func(id, acc, punc int64, nima, niab float32) {
		out.Phoneme = append(out.Phoneme, id)
		out.Accent = append(out.Accent, acc)
		out.Punctuation = append(out.Punctuation, punc)
		out.MoraPosition = append(out.MoraPosition, nima)
		out.PhrasePosition = append(out.PhrasePosition, niab)
	}
	appendPhone(0, 0, 0, 0, 0)
	out.Boundaries = append(out.Boundaries, 0)
	var previous []int
	groupBegin, groupEnd := 0, 0
	for ai, a := range s.Accents {
		if a.Begin < 0 || a.Begin >= len(s.Timeline)-1 || (ai == 0 && a.Begin != 0) || (ai > 0 && a.Begin <= s.Accents[ai-1].Begin) {
			return ProsodyTokens{}, fmt.Errorf("invalid accent phrase start %d", a.Begin)
		}
		end := len(s.Timeline) - 1
		if ai+1 < len(s.Accents) {
			end = s.Accents[ai+1].Begin
		}
		if end <= a.Begin || end > len(s.Timeline)-1 {
			return ProsodyTokens{}, fmt.Errorf("invalid accent phrase extent")
		}
		moraEnd := end
		if s.Timeline[end-1].Type == "pause" {
			moraEnd--
		}
		count := moraEnd - a.Begin
		if count < 1 || a.Up < 0 || a.Up > count || a.Down < 0 || a.Down > count {
			return ProsodyTokens{}, fmt.Errorf("invalid accent positions in phrase %d", ai)
		}
		if len(a.Bound) != 2 || (a.Bound[0] != '|' && a.Bound[0] != '/' && a.Bound[0] != '$') || a.Bound[1] < '0' || a.Bound[1] > '2' {
			return ProsodyTokens{}, fmt.Errorf("unsupported phrase boundary %q", a.Bound)
		}
		if ai >= groupEnd {
			groupBegin = ai
			groupEnd = ai + 1
			for groupEnd < len(s.Accents) {
				start := s.Accents[groupEnd].Begin
				if start <= 0 || start >= len(s.Timeline) {
					return ProsodyTokens{}, fmt.Errorf("invalid accent group extent")
				}
				if s.Timeline[start-1].Type == "pause" {
					break
				}
				groupEnd++
			}
		}
		niab := float32(float64(ai-groupBegin)+0.5) / float32(groupEnd-groupBegin)
		state := int64(1)
		for ti := a.Begin; ti < moraEnd; ti++ {
			item := s.Timeline[ti]
			if item.Type != "mora" {
				return ProsodyTokens{}, fmt.Errorf("unexpected pause within accent phrase")
			}
			units, err := e.EncodePronunciation(item.Mora, false)
			if item.Mora == "ー" {
				units = []MoraUnit{{Text: "ー", PhoneIndices: e.longPhoneIndices(previous), Long: true}}
				err = nil
			}
			if err != nil || len(units) != 1 {
				return ProsodyTokens{}, fmt.Errorf("unsupported mora at timeline item %d: %q", ti, item.Mora)
			}
			mi := ti - a.Begin
			if mi == a.Up-1 {
				state = 3
				if ai > 0 && s.Accents[ai-1].Bound[0] == '/' {
					state = 2
				}
			} else if a.Down > 0 && mi == a.Down {
				state = 1
			}
			nima := float32(float64(mi)+0.5) / float32(count)
			for _, index := range units[0].PhoneIndices {
				appendPhone(int64(e.Phones[index].ID), state, 1, nima, niab)
			}
			if len(units[0].PhoneIndices) > 0 {
				previous = units[0].PhoneIndices
			}
		}
		if moraEnd != end {
			pause := s.Timeline[moraEnd]
			if pause.Pause != "_0" && pause.Pause != "_1" && pause.Pause != "_2" {
				return ProsodyTokens{}, fmt.Errorf("unsupported internal pause %q", pause.Pause)
			}
			out.Boundaries = append(out.Boundaries, len(out.Phoneme))
			appendPhone(0, 0, 0, 0, 0)
		}
	}
	out.Boundaries = append(out.Boundaries, len(out.Phoneme))
	appendPhone(0, 0, endPunc[s.EOS], 0, 0)
	if len(out.Phoneme) > 8192 {
		return ProsodyTokens{}, fmt.Errorf("too many phonemes")
	}
	return out, nil
}
