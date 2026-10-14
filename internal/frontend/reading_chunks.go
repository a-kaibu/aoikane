package frontend

import "fmt"

// ReadingChunk describes the original chunk list, including its end-of-text
// sentinel. Begin/End index the complete word chain, which also includes EOS.
// Subindex == -1 identifies a main chunk; S chunks inherit its dependency.
type ReadingChunk struct {
	Begin    int   `json:"begin"`
	End      int   `json:"end"`
	Ordinal  int32 `json:"ordinal"`
	Subindex int32 `json:"subindex"`
	Link     int32 `json:"link"`
}

// EvaluateChunkContext implements 11e9d0 (P[0:0]) and 11e750 (N[0:0]).
// Previous scopes collect chunks depending on the current ordinal; next scopes
// collect the current chunk's dependency target, including EOS for link -1.
func (r *ReadingRules) EvaluateChunkContext(scope ReadingContext, words []LanguageWord, chunks []ReadingChunk, current int) (bool, error) {
	if scope.Begin != 0 || scope.End != 0 || scope.Direction != "P" && scope.Direction != "N" || current < 0 || current >= len(chunks) {
		return false, fmt.Errorf("invalid chunk reading context")
	}
	for _, c := range chunks {
		if c.Begin < 0 || c.End < c.Begin || c.End > len(words) {
			return false, fmt.Errorf("chunk reading range outside word list")
		}
	}
	link := func(i int) int32 {
		for ; i >= 0; i-- {
			if chunks[i].Subindex == -1 {
				return chunks[i].Link
			}
		}
		return -1
	}
	begin, end, step := current, -1, -1
	target := chunks[current].Ordinal
	if scope.Direction == "N" {
		end, step = len(chunks), 1
		target = link(current)
	}
	for i := begin; i != end; i += step {
		matches := chunks[i].Ordinal == target
		if scope.Direction == "P" {
			matches = link(i) == target
		}
		if !matches {
			continue
		}
		for word := chunks[i].Begin; word < chunks[i].End; word++ {
			// 18e200 accepts EOS when Next is nil or Next.Next is nil:
			// both the sentinel and the last spoken word meet that condition.
			matched, err := r.EvaluateCondition(scope.Condition, words[word], word == 0, word >= len(words)-2)
			if err != nil {
				return false, err
			}
			if matched != scope.All {
				return matched, nil
			}
		}
	}
	return scope.All, nil
}
