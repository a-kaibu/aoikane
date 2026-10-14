package frontend

import (
	"fmt"
	"strings"
)

type ReadingSelection struct {
	Rule    int `json:"rule"`
	Variant int `json:"variant"`
}

// SelectReading reproduces file-order surface/candidate matching followed by
// current, previous, and next conditions. Phrase scopes require a graph
// evaluator; without it they return an error unless legacy skipping is enabled.
func (r *ReadingRules) SelectReading(words []LanguageWord, current int, readings []string) (ReadingSelection, error) {
	return r.selectReading(words, current, readings, nil, nil)
}

// SelectReadingWithContextText also supports N[0:-1] using the exact context
// string produced upstream. The caller must not guess it from neighboring words.
func (r *ReadingRules) SelectReadingWithContextText(words []LanguageWord, current int, readings []string, text string) (ReadingSelection, error) {
	return r.selectReading(words, current, readings, &text, nil)
}

// ReadingScopeEvaluator evaluates P/N[0:0] on the caller's dependency graph.
type ReadingScopeEvaluator func(ReadingContext) (bool, error)

func (r *ReadingRules) SelectReadingWithScopeEvaluator(words []LanguageWord, current int, readings []string, text string, evaluate ReadingScopeEvaluator) (ReadingSelection, error) {
	return r.selectReading(words, current, readings, &text, evaluate)
}

func (r *ReadingRules) selectReading(words []LanguageWord, current int, readings []string, text *string, evaluate ReadingScopeEvaluator) (ReadingSelection, error) {
	if current < 0 || current >= len(words) || len(readings) == 0 {
		return ReadingSelection{}, fmt.Errorf("invalid reading selection input")
	}
	if len(readings) == 1 {
		return ReadingSelection{Rule: -1}, nil
	}
	normalize := func(value string) string {
		// The compact dictionary emits ASCII variant digits; the rule lines
		// use the full-width counterparts. The actual reading is otherwise
		// preserved, including its variant suffix and long-vowel spelling.
		return strings.Map(func(c rune) rune {
			if c >= '０' && c <= '９' {
				return c - '０' + '0'
			}
			return c
		}, value)
	}
	for index, rule := range r.Rules {
		if rule.Surface != words[current].Surface {
			continue
		}
		for variant, reading := range readings {
			if normalize(rule.Reading) != normalize(reading) {
				continue
			}
			matched := true
			for _, direction := range []string{"C", "P", "N"} {
				for _, scope := range rule.Context {
					if scope.Direction != direction {
						continue
					}
					var ok bool
					var err error
					if evaluate != nil && scope.Begin == 0 && scope.End == 0 && (scope.Direction == "P" || scope.Direction == "N") {
						ok, err = evaluate(scope)
					} else if scope.Direction == "N" && scope.Begin == 0 && scope.End == -1 && text != nil {
						ok, err = r.EvaluateWholeCondition(scope.Condition, words, *text, scope.All)
					} else {
						ok, err = r.EvaluateContext(scope, words, current)
					}
					if err != nil {
						return ReadingSelection{}, fmt.Errorf("rule %d: %w", index, err)
					}
					if !ok {
						matched = false
						break
					}
				}
				if !matched {
					break
				}
			}
			if matched {
				return ReadingSelection{Rule: index, Variant: variant}, nil
			}
		}
	}
	return ReadingSelection{Rule: -1}, nil
}
