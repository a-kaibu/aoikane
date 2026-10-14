package frontend

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
)

type EnglishTransliterationRules struct {
	Source  string      `json:"source_sha256"`
	Prefix3 [][2]string `json:"prefix3"`
	Prefix2 [][2]string `json:"prefix2"`
	Prefix1 [][2]string `json:"prefix1"`
	Context [][2]string `json:"context"`
	Single  string      `json:"single"`
	Double  []string    `json:"double"`
	Repeat  string      `json:"repeat"`
	Special string      `json:"special"`
}

func OpenEnglishTransliterationRules(path string) (*EnglishTransliterationRules, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) > 1024*1024 {
		return nil, fmt.Errorf("transliteration table exceeds limit")
	}
	var r EnglishTransliterationRules
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	if r.Source != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(r.Prefix3) != 75 || len(r.Prefix2) != 87 || len(r.Prefix1) != 7 || len(r.Context) != 7 || len(r.Single) != 17 || len(r.Double) != 20 || r.Repeat != "ッ" || r.Special != "ー" {
		return nil, fmt.Errorf("invalid transliteration rules")
	}
	for _, group := range []struct {
		entries [][2]string
		length  int
	}{{r.Prefix3, 3}, {r.Prefix2, 2}, {r.Prefix1, 1}, {r.Context, 1}} {
		for _, pair := range group.entries {
			if len(pair[0]) != group.length || pair[1] == "" {
				return nil, fmt.Errorf("invalid transliteration pair")
			}
			for _, c := range pair[0] {
				if c < 'a' || c > 'z' {
					return nil, fmt.Errorf("invalid transliteration key")
				}
			}
		}
	}
	return &r, nil
}

// Transliterate reconstructs 1a9010. An empty result is the native failure
// result, which the caller must route to the English dictionary pass.
func (r *EnglishTransliterationRules) Transliterate(text string) (string, error) {
	if r == nil || len(text) > 256 {
		return "", fmt.Errorf("invalid transliteration input")
	}
	for _, c := range text {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') {
			return "", fmt.Errorf("transliteration requires ASCII letters")
		}
	}
	var output string
	for pos := 0; pos < len(text); {
		lower := strings.ToLower(text[pos:])
		key, pron := "", ""
		for _, group := range [][][2]string{r.Prefix3, r.Prefix2, r.Prefix1} {
			for _, pair := range group {
				if strings.HasPrefix(lower, pair[0]) {
					key, pron = pair[0], pair[1]
					break
				}
			}
			if key != "" {
				break
			}
		}
		if key != "" {
			if output != "" && pos > 0 {
				for _, pair := range r.Context {
					if pron == pair[1] && text[pos-1] == pair[0][0] {
						pron = r.Special
						break
					}
				}
			}
			output += pron
			pos += len(key)
			continue
		}
		width := 0
		if pos+2 <= len(text) && slices.Contains(r.Double, text[pos:pos+2]) {
			width = 2
		} else if strings.ContainsRune(r.Single, rune(text[pos])) {
			width = 1
		}
		if width == 0 || pos+2*width > len(text) || text[pos:pos+width] != text[pos+width:pos+2*width] || output == "" || strings.HasSuffix(output, r.Repeat) {
			return "", nil
		}
		output += r.Repeat
		pos += width
	}
	if strings.HasPrefix(output, r.Repeat) {
		return "", nil
	}
	return output, nil
}

type EnglishFallbackRules struct {
	Segments *EnglishSegmentRules
	Trees    *EnglishDecisionTrees
	Kana     *EnglishKanaRules
	Codec    *CP932Codec
}

// CompileUnknownEnglish reconstructs 1a4490's spelling, transliteration and
// optional English rule fallback. The fallback is a decision-tree converter,
// not a dictionary search. Earlier morphology supplies dictionary words.
func CompileUnknownEnglish(surface string, spelling *EnglishSpellingRules, transliteration *EnglishTransliterationRules, accent *UnknownAccentRules, fallback ...*EnglishFallbackRules) (UnknownKanaReading, error) {
	if len(fallback) > 1 {
		return UnknownKanaReading{}, fmt.Errorf("multiple English fallback configurations")
	}
	var input strings.Builder
	lower := false
	for _, c := range surface {
		if c >= 'Ａ' && c <= 'Ｚ' {
			input.WriteByte(byte(c - 'Ａ' + 'A'))
		} else if c >= 'ａ' && c <= 'ｚ' {
			input.WriteByte(byte(c - 'ａ' + 'a'))
			lower = true
		} else {
			return UnknownKanaReading{}, fmt.Errorf("English surface is not normalized alphabetic text")
		}
	}
	if input.Len() == 0 {
		return UnknownKanaReading{}, fmt.Errorf("empty English surface")
	}
	if input.Len() <= 3 || input.Len() == 4 && !lower {
		return spelling.Spell(input.String(), accent)
	}
	pron, err := transliteration.Transliterate(input.String())
	if err != nil {
		return UnknownKanaReading{}, err
	}
	if pron == "" {
		if len(fallback) == 0 || fallback[0] == nil {
			return UnknownKanaReading{}, fmt.Errorf("missing English fallback rules for %q", surface)
		}
		f := fallback[0]
		pron, err = f.Segments.ResolveEnglishFallback(input.String(), f.Trees, f.Kana, f.Codec)
		if err != nil {
			return UnknownKanaReading{}, err
		}
	}
	if accent == nil {
		return UnknownKanaReading{}, fmt.Errorf("missing accent rules")
	}
	a, err := accent.Accent(pron)
	if err != nil {
		return UnknownKanaReading{}, err
	}
	return UnknownKanaReading{Pronunciation: pron, Accent: a}, nil
}
