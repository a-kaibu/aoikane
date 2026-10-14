package frontend

import "fmt"

type EnglishInitialCandidate struct {
	Primary        string   `json:"primary"`
	Vowels         []string `json:"vowels"`
	CombinedVowels string   `json:"combined_vowels"`
	Tail           string   `json:"tail"`
	Cost           int      `json:"cost"`
}

// InitialCandidates reconstructs the initial reading/cost pass in 1a6e00
// (1a6eee..1a701a). Context-dependent replacements are a subsequent pass.
func (r *EnglishSegmentRules) InitialCandidates(units []EnglishSegment) ([]EnglishInitialCandidate, error) {
	if r == nil || len(r.Tables) != 6 || len(units) > 30 {
		return nil, fmt.Errorf("invalid English candidate input")
	}
	get := func(ref EnglishRuleRef) (EnglishSegmentRule, error) {
		if ref.Table < 0 || ref.Table >= len(r.Tables) || ref.Index < 0 || ref.Index >= len(r.Tables[ref.Table]) {
			return EnglishSegmentRule{}, fmt.Errorf("invalid English rule reference")
		}
		return r.Tables[ref.Table][ref.Index], nil
	}
	result := make([]EnglishInitialCandidate, len(units))
	for i, unit := range units {
		if len(unit.Vowels) > 2 {
			return nil, fmt.Errorf("English initial candidate exceeds native vowel capacity")
		}
		out := &result[i]
		out.Vowels = make([]string, 0, len(unit.Vowels))
		if unit.Primary != nil {
			rule, err := get(*unit.Primary)
			if err != nil {
				return nil, err
			}
			out.Primary = rule.ClosedReading
			if len(unit.Vowels) > 0 {
				out.Primary = rule.OpenReading
			}
			out.Cost += rule.Cost
		}
		if unit.Tail != nil {
			rule, err := get(*unit.Tail)
			if err != nil {
				return nil, err
			}
			out.Tail = rule.OpenReading
			out.Cost += rule.Cost
		}
		for _, ref := range unit.Vowels {
			rule, err := get(ref)
			if err != nil {
				return nil, err
			}
			out.Vowels = append(out.Vowels, rule.OpenReading)
			out.CombinedVowels += rule.OpenReading
			out.Cost += rule.Cost
		}
	}
	return result, nil
}
