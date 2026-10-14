package frontend

import "fmt"

// LanguageWord supplies the fields used by the language rule predicates. IDs
// are the structured morphology IDs, not the compact dictionary byte codes.
type LanguageWord struct {
	Surface string `json:"surface"`
	Base    string `json:"base"`
	POS     uint32 `json:"pos"`
	Type    uint32 `json:"type"`
	Form    uint32 `json:"form"`
}

// LanguageCondition represents an individual EQ/IN predicate in the reading
// rule table. Position, phrase, BOS/EOS, and compound conditions are separate.
type LanguageCondition struct {
	Field    string   `json:"field"`
	Operator string   `json:"operator"`
	Negate   bool     `json:"negate"`
	Strings  []string `json:"strings,omitempty"`
	Integers []uint32 `json:"integers,omitempty"`
}

func (c LanguageCondition) Evaluate(w LanguageWord) (bool, error) {
	if c.Operator != "EQ" && c.Operator != "IN" {
		return false, fmt.Errorf("unsupported language operator %q", c.Operator)
	}
	matched := false
	switch c.Field {
	case "KANJI", "BASE":
		if len(c.Integers) != 0 || (c.Operator == "EQ" && len(c.Strings) != 1) {
			return false, fmt.Errorf("invalid string language condition")
		}
		value := w.Surface
		if c.Field == "BASE" {
			value = w.Base
		}
		for _, candidate := range c.Strings {
			if value == candidate {
				matched = true
				break
			}
		}
	case "POS", "POS_L1", "POS_L2", "POS_L3", "CTYPE", "CFORM":
		if len(c.Strings) != 0 || (c.Operator == "EQ" && len(c.Integers) != 1) {
			return false, fmt.Errorf("invalid integer language condition")
		}
		value, mask := w.POS, uint32(0xffffffff)
		switch c.Field {
		case "POS_L1":
			mask = 0xff000000
		case "POS_L2":
			mask = 0xffff0000
		case "POS_L3":
			mask = 0xffffff00
		case "CTYPE":
			value = w.Type
		case "CFORM":
			value = w.Form
		}
		value &= mask
		for _, candidate := range c.Integers {
			// Native predicates mask the word's ID, not the rule operand.
			if value == candidate {
				matched = true
				break
			}
		}
	default:
		return false, fmt.Errorf("unsupported language field %q", c.Field)
	}
	return matched != c.Negate, nil
}

func (m *Morphology) LanguageWord(w Word) (LanguageWord, error) {
	resolved, err := m.Resolve(w.Entry)
	if err != nil {
		return LanguageWord{}, err
	}
	return LanguageWord{Surface: w.Text, Base: w.Entry.Surface, POS: resolved.POS.ID,
		Type: resolved.Type.ID, Form: resolved.Form.ID}, nil
}
