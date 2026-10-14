package frontend

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// KanaSentence is the text frontend's accent-bearing output, before contextual
// phonetics and acoustic model controls. Begin counts timeline items, including
// internal pauses; accent positions count moras within each phrase.
type KanaSentence struct {
	BOS      string       `json:"bos"`
	EOS      string       `json:"eos"`
	Timeline []KanaItem   `json:"timeline"`
	Accents  []KanaAccent `json:"accents"`
}

type KanaItem struct {
	Type         string `json:"type"`
	Mora         string `json:"mora,omitempty"`
	Pause        string `json:"pause,omitempty"`
	PhoneIndices []int  `json:"phone_indices,omitempty"`
	Long         bool   `json:"long,omitempty"`
}

type KanaAccent struct {
	Begin int    `json:"begin"`
	Up    int    `json:"accentUp"`
	Down  int    `json:"accentDown"`
	Bound string `json:"bound"`
}

// ParseFormattedKana parses the inspected text2kana format emitted by 1aa4f0.
// It does not determine a reading or combine dictionary accents. Unobserved
// annotations are rejected rather than discarded or guessed.
func (e *MoraEncoder) ParseFormattedKana(text string) (KanaSentence, error) {
	var out KanaSentence
	if !utf8.ValidString(text) || len(text) > 1<<20 || len(text) < 6 || text[0] != '<' || text[2] != '>' || !strings.ContainsRune("SD", rune(text[1])) {
		return out, fmt.Errorf("invalid formatted kana sentence header")
	}
	out.BOS = text[1:2]
	begin, moras, up, down := 0, 0, 0, 0
	var previous []int
	flush := func(bound string) error {
		if moras == 0 || up > moras || down > moras {
			return fmt.Errorf("invalid or empty accent phrase at timeline item %d", begin)
		}
		out.Accents = append(out.Accents, KanaAccent{Begin: begin, Up: up, Down: down, Bound: bound})
		begin = len(out.Timeline)
		moras, up, down = 0, 0, 0
		return nil
	}
	for at := 3; at < len(text); {
		switch text[at] {
		case '^':
			if up != 0 {
				return KanaSentence{}, fmt.Errorf("duplicate rise at byte %d", at)
			}
			up = moras + 1
			at++
		case '!':
			if down != 0 || moras == 0 {
				return KanaSentence{}, fmt.Errorf("invalid fall at byte %d", at)
			}
			down = moras
			at++
		case '|', '/', '$':
			if at+1 >= len(text) || text[at+1] < '0' || text[at+1] > '2' {
				return KanaSentence{}, fmt.Errorf("unsupported boundary at byte %d", at)
			}
			if err := flush(text[at : at+2]); err != nil {
				return KanaSentence{}, err
			}
			at += 2
		case '_':
			if moras != 0 || up != 0 || at+1 >= len(text) || text[at+1] < '0' || text[at+1] > '2' {
				return KanaSentence{}, fmt.Errorf("unsupported pause at byte %d", at)
			}
			out.Timeline = append(out.Timeline, KanaItem{Type: "pause", Pause: text[at : at+2]})
			begin = len(out.Timeline)
			at += 2
		case '<':
			if at+3 != len(text) || text[at+2] != '>' || !strings.ContainsRune("FHRAN", rune(text[at+1])) {
				return KanaSentence{}, fmt.Errorf("unsupported sentence end at byte %d", at)
			}
			if err := flush("|0"); err != nil {
				return KanaSentence{}, err
			}
			out.EOS = text[at+1 : at+2]
			out.Timeline = append(out.Timeline, KanaItem{Type: "pause", Pause: "eos"})
			return out, nil
		default:
			indices, n := e.LookupPrefix(text[at:], false)
			long := strings.HasPrefix(text[at:], "ー")
			if long {
				indices, n = e.longPhoneIndices(previous), len("ー")
			}
			if n == 0 {
				return KanaSentence{}, fmt.Errorf("unsupported formatted kana at byte %d", at)
			}
			out.Timeline = append(out.Timeline, KanaItem{Type: "mora", Mora: text[at : at+n], PhoneIndices: indices, Long: long})
			if len(indices) > 0 {
				previous = indices
			}
			moras++
			at += n
		}
	}
	return KanaSentence{}, fmt.Errorf("missing formatted kana sentence end")
}
