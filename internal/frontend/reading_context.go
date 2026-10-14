package frontend

import "fmt"

// EvaluateContext handles the current word and the inspected word-range
// branches of 11e3b0. P/N[0:0] use EvaluateChunkContext instead; N[0:-1]
// uses EvaluateWholeCondition. This flat-word entry point has neither context.
func (r *ReadingRules) EvaluateContext(scope ReadingContext, words []LanguageWord, current int) (bool, error) {
	if len(words) == 0 || len(words) > 8192 || current < 0 || current >= len(words) ||
		scope.Begin < -32768 || scope.Begin > 32767 || scope.End < -32768 || scope.End > 32767 {
		return false, fmt.Errorf("invalid reading context")
	}
	evaluate := func(index int) (bool, error) {
		return r.EvaluateCondition(scope.Condition, words[index], index == 0, index == len(words)-1)
	}
	if scope.Direction == "C" {
		return evaluate(current)
	}
	if scope.Direction != "P" && scope.Direction != "N" {
		return false, fmt.Errorf("unsupported context direction %q", scope.Direction)
	}
	if scope.Begin == 0 && scope.End == 0 || scope.Direction == "N" && scope.Begin == 0 && scope.End == -1 {
		if r.SkipPhraseScopes {
			return false, nil
		}
		return false, fmt.Errorf("phrase reading scope %s[%d:%d] is not reconstructed", scope.Direction, scope.Begin, scope.End)
	}
	// The native immediate-neighbor branches ignore All and require a neighbor.
	if scope.Begin == 1 && scope.End == 1 {
		index := current - 1
		if scope.Direction == "N" {
			index = current + 1
		}
		if index < 0 || index >= len(words) {
			return false, nil
		}
		return evaluate(index)
	}
	var begin, end int
	if scope.Direction == "P" {
		upper := current - scope.Begin
		if scope.Begin < 0 {
			upper = -scope.Begin - 1
		}
		lower := current - scope.End
		if scope.End <= 0 {
			lower = -scope.End - 1
		}
		lower = max(-1, min(lower, upper-1))
		if upper < 0 {
			return scope.All, nil
		}
		begin, end = lower+1, upper+1
	} else {
		// Other zero-start next scopes are a no-op in the inspected branch.
		if scope.Begin == 0 {
			return true, nil
		}
		begin = scope.Begin
		if begin > 0 {
			begin += current
		} else {
			begin += len(words)
		}
		end = scope.End
		if end > 0 {
			end += current
		} else {
			end += len(words) + 1
		}
		end = max(begin+1, end)
		if begin >= len(words) {
			return scope.All, nil
		}
		end = min(end, len(words))
	}
	if begin < 0 || end > len(words) {
		return false, fmt.Errorf("reading range outside word list")
	}
	for i := begin; i < end; i++ {
		matched, err := evaluate(i)
		if err != nil {
			return false, err
		}
		if matched && !scope.All {
			return true, nil
		}
		if !matched && scope.All {
			return false, nil
		}
	}
	return scope.All, nil
}
