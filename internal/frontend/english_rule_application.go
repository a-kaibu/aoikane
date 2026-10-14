package frontend

import "fmt"

// ApplyEnglishRuleResults reconstructs 1a6760 after the four decision-tree
// evaluations. '?' preserves the existing field; an empty result clears it.
// Decision-tree traversal is a separate stage and must supply actual results.
func ApplyEnglishRuleResults(candidate EnglishInitialCandidate, results [4]string) (EnglishInitialCandidate, error) {
	for _, s := range results {
		if len(s) > 4096 {
			return EnglishInitialCandidate{}, fmt.Errorf("English rule result exceeds limit")
		}
		for _, c := range s {
			if c == 0 || c > 127 {
				return EnglishInitialCandidate{}, fmt.Errorf("English rule result requires non-NUL ASCII")
			}
		}
	}
	if results[0] != "?" {
		candidate.Primary = results[0]
	}
	if results[1] != "?" {
		candidate.CombinedVowels = results[1]
	}
	if results[2] != "?" {
		candidate.Tail = results[2]
	}
	if results[3] != "?" {
		candidate.Cost = 0
		if results[3] == "1" {
			candidate.Cost = 1
		}
	}
	return candidate, nil
}
