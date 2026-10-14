package frontend

import "fmt"

type EnglishContextUnit struct {
	Primary     string `json:"primary"`
	VowelPrefix string `json:"vowel_prefix"`
	VowelTail   string `json:"vowel_tail"`
	Combined    string `json:"combined"`
	Mask        int    `json:"mask"`
}

type EnglishConditionContext struct {
	// Native order: current, next, previous.
	Units         [3]EnglishContextUnit `json:"units"`
	Following     int                   `json:"following"`
	Index         int                   `json:"index"`
	LastCharacter string                `json:"last_character"`
	OptionA       int                   `json:"option_a"`
	OptionB       int                   `json:"option_b"`
}

// BuildEnglishConditionContext reconstructs 1a81f0 from prepared unit
// strings. Empty fields use the native '_' placeholder, while concatenation
// includes only actual fields. A missing neighbor also has combined '_'.
func BuildEnglishConditionContext(units []PreparedEnglishSegment, index, optionA, optionB int) (EnglishConditionContext, error) {
	if len(units) == 0 || len(units) > 30 || index < 0 || index >= len(units) {
		return EnglishConditionContext{}, fmt.Errorf("invalid English context scope")
	}
	for _, unit := range units {
		for _, s := range []string{unit.Primary, unit.VowelPrefix, unit.VowelTail} {
			if len(s) > 128 {
				return EnglishConditionContext{}, fmt.Errorf("English context string exceeds limit")
			}
			for _, c := range s {
				if c == 0 || c > 127 {
					return EnglishConditionContext{}, fmt.Errorf("English context requires non-NUL ASCII")
				}
			}
		}
	}
	result := EnglishConditionContext{Following: len(units) - index - 1, Index: index, OptionA: optionA, OptionB: optionB, LastCharacter: "_"}
	for i, position := range [3]int{index, index + 1, index - 1} {
		u := EnglishContextUnit{Primary: "_", VowelPrefix: "_", VowelTail: "_", Combined: "_"}
		if position >= 0 && position < len(units) {
			source := units[position]
			u.Combined = source.Primary + source.VowelPrefix + source.VowelTail
			if source.Primary != "" {
				u.Primary = source.Primary
				u.Mask |= 1
			}
			if source.VowelPrefix != "" {
				u.VowelPrefix = source.VowelPrefix
				u.Mask |= 2
			}
			if source.VowelTail != "" {
				u.VowelTail = source.VowelTail
				u.Mask |= 4
			}
		}
		result.Units[i] = u
	}
	last := units[len(units)-1]
	for _, s := range []string{last.VowelTail, last.VowelPrefix, last.Primary} {
		if len(s) > 0 {
			result.LastCharacter = s[len(s)-1:]
			break
		}
	}
	return result, nil
}
