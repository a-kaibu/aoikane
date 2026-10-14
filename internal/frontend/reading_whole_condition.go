package frontend

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// EvaluateWholeCondition reproduces the predicate virtual method used for
// N[0:-1]. The context text is supplied by the upstream language stage; it is
// not synthesized by concatenating the following words.
func (r *ReadingRules) EvaluateWholeCondition(id int, words []LanguageWord, text string, all bool) (bool, error) {
	if len(words) > 8192 || len(text) > 1<<20 || !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
		return false, fmt.Errorf("invalid whole reading context")
	}
	p, ok := r.Conditions[id]
	if !ok {
		return false, fmt.Errorf("missing CD:%d", id)
	}
	if p.Kind == "and" {
		// This class's whole-context virtual method is the constant-true
		// function b9510, rather than the per-word conjunction evaluator.
		return true, nil
	}
	if p.Kind != "word" || (p.Condition.Field != "KANJI" && p.Condition.Field != "BASE") {
		return false, fmt.Errorf("unsupported whole-context predicate %q", p.Kind)
	}
	if _, err := p.Condition.Evaluate(LanguageWord{}); err != nil {
		return false, err
	}
	if p.Condition.Field == "KANJI" {
		matched := false
		// Native search tests positions before the text's NUL terminator.
		// Even an empty pattern fails against an empty context string.
		if text != "" {
			for _, pattern := range p.Condition.Strings {
				if strings.Contains(pattern, "\x00") || !utf8.ValidString(pattern) {
					return false, fmt.Errorf("invalid reading search pattern")
				}
				if strings.Contains(text, pattern) {
					matched = true
					break
				}
			}
		}
		// The text-search branch ignores All, and negates the final match.
		return matched != p.Condition.Negate, nil
	}
	for _, word := range words {
		matched, err := p.Condition.Evaluate(word)
		if err != nil {
			return false, err
		}
		if matched && !all {
			return true, nil
		}
		if !matched && all {
			return false, nil
		}
	}
	return all, nil
}
