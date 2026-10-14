package frontend

import (
	"encoding/json"
	"fmt"
	"os"
	"unicode/utf8"
)

// UnknownKanaRules reconstructs 1a58c0's pronunciation conversion, not the
// preceding unknown-word grouping or the following accent assignment.
type UnknownKanaRules struct {
	small   map[rune]rune
	pairs   map[string]bool
	triples map[string]bool
}

func OpenUnknownKanaRules(path string) (*UnknownKanaRules, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) > 1024*1024 {
		return nil, fmt.Errorf("unknown-kana table exceeds limit")
	}
	var table struct {
		Source  string      `json:"source_sha256"`
		Small   [][2]string `json:"small"`
		Pairs   []string    `json:"pairs"`
		Triples []string    `json:"triples"`
	}
	if err := json.Unmarshal(b, &table); err != nil {
		return nil, err
	}
	if table.Source != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(table.Small) == 0 {
		return nil, fmt.Errorf("invalid unknown-kana table")
	}
	r := &UnknownKanaRules{small: make(map[rune]rune), pairs: make(map[string]bool), triples: make(map[string]bool)}
	for _, entry := range table.Small {
		a, b := []rune(entry[0]), []rune(entry[1])
		if len(a) != 1 || len(b) != 1 {
			return nil, fmt.Errorf("invalid small kana mapping")
		}
		if _, exists := r.small[a[0]]; exists {
			return nil, fmt.Errorf("duplicate small kana mapping")
		}
		r.small[a[0]] = b[0]
	}
	for _, s := range table.Pairs {
		if utf8.RuneCountInString(s) != 2 {
			return nil, fmt.Errorf("invalid unknown-kana pair")
		}
		r.pairs[s] = true
	}
	for _, s := range table.Triples {
		if utf8.RuneCountInString(s) != 3 {
			return nil, fmt.Errorf("invalid unknown-kana triple")
		}
		r.triples[s] = true
	}
	return r, nil
}

func (r *UnknownKanaRules) Pronunciation(text string) (string, error) {
	if len(text) > 4096 || !utf8.ValidString(text) {
		return "", fmt.Errorf("invalid unknown-kana text")
	}
	input := []rune(text)
	for _, c := range input {
		if !(c >= 0x30a1 && c <= 0x30f6 || c == 0x30fc || c == 0x309c) {
			return "", fmt.Errorf("unknown pronunciation for non-katakana U+%04X is not reconstructed", c)
		}
	}
	output := append([]rune(nil), input...)
	if len(input) > 0 {
		if expanded, small := r.small[input[0]]; small {
			output[0] = expanded
		}
	}
	for i := 1; i < len(input); i++ {
		expanded, small := r.small[input[i]]
		if !small {
			continue
		}
		if i+1 < len(input) {
			if _, nextSmall := r.small[input[i+1]]; nextSmall && r.triples[string(input[i-1:i+2])] {
				i++
				continue
			}
		}
		if !r.pairs[string(input[i-1:i+1])] {
			output[i] = expanded
		}
	}
	return string(output), nil
}
