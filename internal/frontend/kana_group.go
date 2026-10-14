package frontend

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// KanaGroupWord preserves the source-origin vector size used by the native
// first pass. Its group boundary must come from the document graph.
type KanaGroupWord struct {
	KanaMergeWord
	OriginCount int `json:"origin_count"`
}

// KanaRun indexes the original group, with an exclusive end.
type KanaRun struct {
	Begin int `json:"begin"`
	End   int `json:"end"`
}

// SelectKanaRuns reconstructs the first pass of 1a4770 within one explicit
// document group. An origin-bearing kana word starts a new run; non-kana words
// end the current run. A single-word run is still passed to regeneration.
func SelectKanaRuns(words []KanaGroupWord) ([]KanaRun, error) {
	if len(words) > 8192 {
		return nil, fmt.Errorf("kana group exceeds limit")
	}
	var runs []KanaRun
	begin := -1
	flush := func(end int) {
		if begin >= 0 {
			runs = append(runs, KanaRun{Begin: begin, End: end})
			begin = -1
		}
	}
	for i, word := range words {
		if word.OriginCount < 0 || len(word.Surface) > 1024*1024 || !utf8.ValidString(word.Surface) || strings.ContainsRune(word.Surface, 0) {
			return nil, fmt.Errorf("invalid kana group word %d", i)
		}
		isKana := word.Surface != ""
		for _, c := range word.Surface {
			if !(c >= 0x30a1 && c <= 0x30f6 || c == 0x30fc) {
				isKana = false
				break
			}
		}
		if !isKana {
			flush(i)
			continue
		}
		if word.OriginCount > 0 {
			flush(i)
		}
		if begin < 0 {
			begin = i
		}
	}
	flush(len(words))
	return runs, nil
}

// MergeKanaGroup applies run selection and regeneration without processing
// later ASCII grouping or word cleanup passes.
func MergeKanaGroup(words []KanaGroupWord, codec *CP932Codec, kana *UnknownKanaRules, accent *UnknownAccentRules) ([]KanaMergeWord, error) {
	runs, err := SelectKanaRuns(words)
	if err != nil {
		return nil, err
	}
	input := make([]KanaMergeWord, len(words))
	for i, word := range words {
		input[i] = word.KanaMergeWord
	}
	result := make([]KanaMergeWord, 0, len(words))
	pos := 0
	for _, run := range runs {
		result = append(result, input[pos:run.Begin]...)
		merged, err := MergeKanaRun(input[run.Begin:run.End], codec, kana, accent)
		if err != nil {
			return nil, err
		}
		result = append(result, merged...)
		pos = run.End
	}
	return append(result, input[pos:]...), nil
}
