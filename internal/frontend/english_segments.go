package frontend

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type EnglishSegmentRule struct {
	Pattern       string `json:"pattern"`
	OpenReading   string `json:"open_reading"`
	ClosedReading string `json:"closed_reading"`
	Cost          int    `json:"cost"`
	Weight        int    `json:"weight"`
}

type EnglishRuleRef struct {
	Table int `json:"table"`
	Index int `json:"index"`
}

type EnglishSegment struct {
	Primary *EnglishRuleRef  `json:"primary"`
	Vowels  []EnglishRuleRef `json:"vowels"`
	Tail    *EnglishRuleRef  `json:"tail"`
	Weight  int              `json:"weight"`
}

type EnglishSegmentRules struct {
	Source string                 `json:"source_sha256"`
	Tables [][]EnglishSegmentRule `json:"tables"`
}

type PreparedEnglishSegment struct {
	Primary     string `json:"primary"`
	VowelPrefix string `json:"vowel_prefix"`
	VowelTail   string `json:"vowel_tail"`
}

// Prepare reconstructs 1a8720: split the original letter patterns into the
// strings used by later pronunciation conditions. It does not choose phonemes.
func (r *EnglishSegmentRules) Prepare(units []EnglishSegment) ([]PreparedEnglishSegment, error) {
	if r == nil || len(r.Tables) != 6 || len(units) > 30 {
		return nil, fmt.Errorf("invalid English preparation input")
	}
	get := func(ref EnglishRuleRef) (EnglishSegmentRule, error) {
		if ref.Table < 0 || ref.Table >= len(r.Tables) || ref.Index < 0 || ref.Index >= len(r.Tables[ref.Table]) {
			return EnglishSegmentRule{}, fmt.Errorf("invalid English rule reference")
		}
		return r.Tables[ref.Table][ref.Index], nil
	}
	result := make([]PreparedEnglishSegment, len(units))
	for i, unit := range units {
		if len(unit.Vowels) > 3 {
			return nil, fmt.Errorf("English segment has too many vowels")
		}
		out := &result[i]
		if unit.Primary != nil {
			rule, err := get(*unit.Primary)
			if err != nil {
				return nil, err
			}
			out.Primary = rule.Pattern
			if rule.Weight > 0 {
				if rule.Weight > len(rule.Pattern) {
					return nil, fmt.Errorf("invalid primary suffix weight")
				}
				end := len(rule.Pattern) - rule.Weight
				if end != 0 {
					out.VowelPrefix = rule.Pattern[end:]
					out.Primary = rule.Pattern[:end]
				}
			}
		}
		for _, ref := range unit.Vowels {
			rule, err := get(ref)
			if err != nil {
				return nil, err
			}
			if rule.Weight < 0 || rule.Weight > len(rule.Pattern) {
				return nil, fmt.Errorf("invalid vowel weight")
			}
			out.VowelPrefix += rule.Pattern[:rule.Weight]
			out.VowelTail += rule.Pattern[rule.Weight:]
		}
		if unit.Tail != nil {
			rule, err := get(*unit.Tail)
			if err != nil {
				return nil, err
			}
			out.VowelTail += rule.Pattern
		}
	}
	return result, nil
}

func OpenEnglishSegmentRules(path string) (*EnglishSegmentRules, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) > 1024*1024 {
		return nil, fmt.Errorf("English segmentation rules exceed limit")
	}
	var r EnglishSegmentRules
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	if r.Source != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(r.Tables) != 6 {
		return nil, fmt.Errorf("invalid English segmentation rules")
	}
	for i, count := range []int{32, 6, 5, 68, 4, 21} {
		if len(r.Tables[i]) != count {
			return nil, fmt.Errorf("unexpected English segmentation table %d", i)
		}
		for _, rule := range r.Tables[i] {
			if rule.Pattern == "" || rule.Weight < 0 || rule.Weight > 3 {
				return nil, fmt.Errorf("invalid English segment rule")
			}
		}
	}
	return &r, nil
}

// Segment reconstructs 1a7af0 after the first-chunk input preparation of
// 1a6e00. Subsequent chunks and phonetic rule selection are separate passes.
func (r *EnglishSegmentRules) Segment(text string) ([]EnglishSegment, error) {
	if r == nil || len(r.Tables) != 6 || len(text) > 1024*1024 {
		return nil, fmt.Errorf("invalid English segmentation input")
	}
	for _, c := range text {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') {
			return nil, fmt.Errorf("English segmentation requires ASCII letters")
		}
	}
	input := "^"
	if len(text) < 30 {
		input += strings.ToLower(text) + "$"
	} else {
		input += strings.ToLower(text[:30])
	}
	units := make([]EnglishSegment, 32)
	for i := range units {
		units[i].Vowels = make([]EnglishRuleRef, 0)
	}
	current := 0
	match := func(table, pos int) int {
		for i, rule := range r.Tables[table] {
			if strings.HasPrefix(input[pos:], rule.Pattern) {
				return i
			}
		}
		return -1
	}
	for pos := 0; pos < len(input) && pos < 31 && input[pos] != '$'; {
		if current >= 30 {
			return nil, fmt.Errorf("English segmentation exceeded native workspace")
		}
		table, index := -1, -1
		for t := 0; t < 4; t++ {
			if i := match(t, pos); i >= 0 {
				table, index = t, i
				break
			}
		}
		if table < 0 && len(units[current].Vowels) > 0 {
			if i := match(4, pos); i >= 0 {
				next := pos + len(r.Tables[4][i].Pattern)
				if next <= len(input) && match(2, next) < 0 {
					table, index = 4, i
				}
			}
		}
		if table < 0 {
			if i := match(5, pos); i >= 0 {
				table, index = 5, i
			}
		}
		if table < 0 {
			pos++
			continue
		}
		rule := r.Tables[table][index]
		ref := EnglishRuleRef{Table: table, Index: index}
		switch table {
		case 0:
			if units[current].Primary != nil || len(units[current].Vowels) > 0 {
				current++
			}
			units[current].Primary = &ref
			current++
		case 1, 2:
			if units[current].Weight+rule.Weight > 2 {
				current++
			}
			units[current].Vowels = append(units[current].Vowels, ref)
			units[current].Weight += rule.Weight
			if table == 1 {
				current++
			}
		case 3, 5:
			if units[current].Primary != nil || len(units[current].Vowels) > 0 {
				current++
			}
			units[current].Primary = &ref
			units[current].Weight += rule.Weight
		case 4:
			units[current].Tail = &ref
			units[current].Weight += rule.Weight
			current++
		}
		pos += len(rule.Pattern)
	}
	if units[current].Primary != nil || len(units[current].Vowels) > 0 {
		current++
	}
	return units[:current], nil
}
