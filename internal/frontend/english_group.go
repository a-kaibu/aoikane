package frontend

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type EnglishGroupWord struct {
	KanaGroupWord
	// Supplied by the preceding source-origin context pass (native +0x30 == 1).
	BoundaryBefore bool `json:"boundary_before"`
}

type EnglishGroupSelection struct {
	Runs []KanaRun `json:"runs"`
	End  int       `json:"end"` // number of original words consumed by one scan
}

// SelectEnglishRuns reconstructs one 1a4c40 scan. Non-alphabetic words end the
// scan after being consumed; an origin-context boundary stops before its word.
// A non-boundary origin-bearing alphabetic word starts a new run.
func SelectEnglishRuns(words []EnglishGroupWord) (EnglishGroupSelection, error) {
	out := EnglishGroupSelection{Runs: make([]KanaRun, 0)}
	if len(words) > 8192 {
		return out, fmt.Errorf("English group exceeds limit")
	}
	begin := -1
	flush := func(end int) {
		if begin >= 0 {
			out.Runs = append(out.Runs, KanaRun{Begin: begin, End: end})
			begin = -1
		}
	}
	for i, w := range words {
		if w.OriginCount < 0 || len(w.Surface) > 1024*1024 || !utf8.ValidString(w.Surface) || strings.ContainsRune(w.Surface, 0) {
			return out, fmt.Errorf("invalid English group word %d", i)
		}
		if w.BoundaryBefore {
			flush(i)
			out.End = i
			return out, nil
		}
		alphabetic := w.Surface != ""
		for _, c := range w.Surface {
			if !(c >= 'Ａ' && c <= 'Ｚ' || c >= 'ａ' && c <= 'ｚ') {
				alphabetic = false
				break
			}
		}
		if !alphabetic {
			flush(i)
			out.End = i + 1
			return out, nil
		}
		if w.OriginCount > 0 {
			flush(i)
		}
		if begin < 0 {
			begin = i
		}
	}
	flush(len(words))
	out.End = len(words)
	return out, nil
}

func MergeEnglishGroupScan(words []EnglishGroupWord, codec *CP932Codec, read EnglishReadingFunc) ([]KanaMergeWord, int, error) {
	selection, err := SelectEnglishRuns(words)
	if err != nil {
		return nil, 0, err
	}
	input := make([]KanaMergeWord, selection.End)
	for i := range input {
		input[i] = words[i].KanaMergeWord
	}
	out := make([]KanaMergeWord, 0, len(input))
	position := 0
	for _, run := range selection.Runs {
		out = append(out, input[position:run.Begin]...)
		merged, err := MergeEnglishRun(input[run.Begin:run.End], codec, read)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, merged...)
		position = run.End
	}
	out = append(out, input[position:]...)
	return out, selection.End, nil
}
