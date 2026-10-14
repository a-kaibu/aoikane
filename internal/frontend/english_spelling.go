package frontend

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type EnglishSpellingLetter struct {
	Letter        string `json:"letter"`
	Pronunciation string `json:"pronunciation"`
	Down          int    `json:"down"`
}

type EnglishSpellingRules struct {
	Source  string                  `json:"source_sha256"`
	Letters []EnglishSpellingLetter `json:"letters"`
}

func OpenEnglishSpellingRules(path string) (*EnglishSpellingRules, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) > 1024*1024 {
		return nil, fmt.Errorf("spelling rules exceed limit")
	}
	var rules EnglishSpellingRules
	if err := json.Unmarshal(b, &rules); err != nil {
		return nil, err
	}
	if rules.Source != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(rules.Letters) != 26 {
		return nil, fmt.Errorf("invalid spelling rules")
	}
	for i, letter := range rules.Letters {
		if letter.Letter != string(rune('a'+i)) || letter.Pronunciation == "" || letter.Down < 0 {
			return nil, fmt.Errorf("invalid spelling letter %d", i)
		}
	}
	return &rules, nil
}

// Spell implements 1a8e80 for ASCII input. Earlier unrecognized bytes are
// skipped, but an unrecognized final byte clears the native result.
// Accent is based on the last letter's table entry, not unknown-word accent.
func (r *EnglishSpellingRules) Spell(text string, accent *UnknownAccentRules) (UnknownKanaReading, error) {
	if r == nil || len(r.Letters) != 26 || accent == nil || len(text) > 256 {
		return UnknownKanaReading{}, fmt.Errorf("invalid spelling input")
	}
	var output strings.Builder
	last := 0 // The native empty-input path uses the first table entry.
	for i := range text {
		c := text[i]
		if c == 0 || c > 127 {
			return UnknownKanaReading{}, fmt.Errorf("spelling requires non-NUL ASCII")
		}
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		last = int(c) - 'a'
		if last >= 0 && last < 26 {
			output.WriteString(r.Letters[last].Pronunciation)
		} else if i == len(text)-1 {
			return UnknownKanaReading{Accent: Accent{Down: -1}}, nil
		}
	}
	pronunciation := output.String()
	total, err := accent.Accent(pronunciation)
	if err != nil {
		return UnknownKanaReading{}, err
	}
	final, err := accent.Accent(r.Letters[last].Pronunciation)
	if err != nil {
		return UnknownKanaReading{}, err
	}
	return UnknownKanaReading{Pronunciation: pronunciation, Accent: Accent{
		Moras: total.Moras, Down: total.Moras - final.Moras + r.Letters[last].Down,
	}}, nil
}

// CompileEnglishSpelling reconstructs the spelling branch of 1a4490 for a
// normalized, full-width alphabetic surface. Other branches require the
// separate transliteration and English dictionary passes.
func (r *EnglishSpellingRules) CompileEnglishSpelling(surface string, accent *UnknownAccentRules) (UnknownKanaReading, error) {
	var input strings.Builder
	lower := false
	for _, c := range surface {
		switch {
		case c >= 'Ａ' && c <= 'Ｚ':
			input.WriteByte(byte(c - 'Ａ' + 'A'))
		case c >= 'ａ' && c <= 'ｚ':
			input.WriteByte(byte(c - 'ａ' + 'a'))
			lower = true
		default:
			return UnknownKanaReading{}, fmt.Errorf("English surface is not normalized alphabetic text")
		}
	}
	if input.Len() == 0 || input.Len() > 4 || input.Len() == 4 && lower {
		return UnknownKanaReading{}, fmt.Errorf("English transliteration or dictionary lookup is not reconstructed for %q", surface)
	}
	return r.Spell(input.String(), accent)
}
