package frontend

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"unicode/utf8"
)

type UnknownAccentRules struct {
	Source   string   `json:"source_sha256"`
	Small    []string `json:"small"`
	Ignore   []string `json:"ignore"`
	Suffix3  []string `json:"suffix3"`
	Suffix2  []string `json:"suffix2"`
	Endings1 []string `json:"endings1"`
	Endings2 []string `json:"endings2"`
	Tail     []string `json:"tail"`
	Special  []string `json:"special"`
}

type UnknownKanaReading struct {
	Pronunciation string `json:"pronunciation"`
	Accent        Accent `json:"accent"`
}

// CompileUnknownKana joins the inspected pronunciation and accent passes.
// Word grouping, morphology, property normalization, and phrase merging follow
// different contracts and are not inferred here.
func CompileUnknownKana(text string, pronunciation *UnknownKanaRules, accent *UnknownAccentRules) (UnknownKanaReading, error) {
	if pronunciation == nil || accent == nil {
		return UnknownKanaReading{}, fmt.Errorf("missing unknown-kana rules")
	}
	reading, err := pronunciation.Pronunciation(text)
	if err != nil {
		return UnknownKanaReading{}, err
	}
	a, err := accent.Accent(reading)
	if err != nil {
		return UnknownKanaReading{}, err
	}
	return UnknownKanaReading{Pronunciation: reading, Accent: a}, nil
}

func OpenUnknownAccentRules(path string) (*UnknownAccentRules, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) > 1024*1024 {
		return nil, fmt.Errorf("unknown accent table exceeds limit")
	}
	var r UnknownAccentRules
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	if r.Source != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(r.Tail) != 2 || len(r.Special) != 6 || len(r.Small) != 10 {
		return nil, fmt.Errorf("invalid unknown accent table")
	}
	for _, group := range [][]string{r.Small, r.Ignore, r.Tail, r.Special} {
		for _, s := range group {
			if !utf8.ValidString(s) || utf8.RuneCountInString(s) != 1 {
				return nil, fmt.Errorf("invalid accent character")
			}
		}
	}
	return &r, nil
}

// Accent implements 1962d0 and 1a8b10 for the inspected kana input domain.
// The native fall position is assigned separately from phrase accent merging.
func (r *UnknownAccentRules) Accent(text string) (Accent, error) {
	if !utf8.ValidString(text) || len(text) > 4096 {
		return Accent{}, fmt.Errorf("invalid unknown accent input")
	}
	chars := []rune(text)
	for _, c := range chars {
		if !(c >= 0x30a1 && c <= 0x30f6 || c == 0x30fc || c == 0x309c || c == 0x3001 || c == 0x3002) {
			return Accent{}, fmt.Errorf("unknown accent for U+%04X is not reconstructed", c)
		}
	}
	isSmall := func(c rune) bool { return slices.Contains(r.Small, string(c)) }
	count := 0
	for i := 0; i < len(chars); {
		if slices.Contains(r.Ignore, string(chars[i])) {
			i++
			continue
		}
		first := chars[i]
		i++
		count++
		if !isSmall(first) && i < len(chars) && isSmall(chars[i]) {
			i++
			if i < len(chars) && isSmall(chars[i]) {
				i++
			}
		}
	}
	result := Accent{Down: 1, Moras: count}
	if count < 4 {
		return result, nil
	}
	pos, adjust := len(chars), 0
	if count > 4 {
		matched := false
		for _, s := range r.Suffix3 {
			if strings.HasSuffix(text, s) {
				pos = len(chars) - 3
				adjust = 2
				matched = true
				break
			}
		}
		if !matched {
			for _, s := range r.Suffix2 {
				if strings.HasSuffix(text, s) {
					pos = len(chars) - 3
					adjust = 1
					matched = true
					break
				}
			}
		}
		if !matched {
			for _, s := range r.Endings1 {
				if strings.HasSuffix(text, s) {
					pos = len(chars) - 2
					adjust = 1
					matched = true
					break
				}
			}
		}
		if !matched {
			for _, s := range r.Endings2 {
				if strings.HasSuffix(text, s) {
					pos = len(chars) - 2
					adjust = 2
					break
				}
			}
		}
	}
	pos--
	at := func(i int) string {
		if i < 0 || i >= len(chars) {
			return ""
		}
		return string(chars[i])
	}
	tailCount := 1
	if count == 4 {
		tailCount = 2
	}
	extra := 0
	if slices.Contains(r.Tail[:tailCount], at(pos)) {
		extra = 1
		pos--
	}
	penalty := adjust + extra
	for regular := 0; pos >= 0 && regular < 2; pos-- {
		if isSmall(chars[pos]) {
			continue
		}
		if slices.Contains(r.Special, at(pos)) {
			extra = 1
		}
		regular++
		penalty++
	}
	if count == 4 {
		if slices.Contains(r.Special, at(pos)) {
			penalty++
		} else if extra == 0 {
			result.Down = 0
			return result, nil
		}
		result.Down = 4 - penalty
		return result, nil
	}
	if !slices.Contains(r.Special[:1], at(pos+1)) && pos > 0 && slices.Contains(r.Special, at(pos)) {
		penalty++
		if pos > 1 && slices.Contains(r.Special[:2], at(pos-1)) {
			penalty++
		}
	}
	result.Down = max(1, count-penalty)
	return result, nil
}
