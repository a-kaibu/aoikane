package frontend

import (
	"fmt"
	"path/filepath"
	"strings"
)

// EnglishReader assembles the exported spelling, transliteration and
// decision-tree rules used by the native unknown-English conversion.
type EnglishReader struct {
	Spelling        *EnglishSpellingRules
	Transliteration *EnglishTransliterationRules
	Accent          *UnknownAccentRules
	Fallback        *EnglishFallbackRules
}

func OpenEnglishReader(dir string, codec *CP932Codec, accent *UnknownAccentRules) (*EnglishReader, error) {
	path := func(name string) string { return filepath.Join(dir, name+".json") }
	r := &EnglishReader{Accent: accent}
	var err error
	if r.Spelling, err = OpenEnglishSpellingRules(path("english-spelling")); err != nil {
		return nil, err
	}
	if r.Transliteration, err = OpenEnglishTransliterationRules(path("english-transliteration")); err != nil {
		return nil, err
	}
	f := &EnglishFallbackRules{Codec: codec}
	if f.Segments, err = OpenEnglishSegmentRules(path("english-segments")); err != nil {
		return nil, err
	}
	if f.Trees, err = OpenEnglishDecisionTrees(path("english-decisions")); err != nil {
		return nil, err
	}
	if f.Kana, err = OpenEnglishKanaRules(path("english-kana")); err != nil {
		return nil, err
	}
	r.Fallback = f
	return r, nil
}

// ConventionalEnglishPronunciation supplies exact-word corrections for both
// dictionary words and guessed readings. Input is normalized full-width text.
// Never replace substrings: Alice must not change malice or related spellings.
func ConventionalEnglishPronunciation(surface string) string {
	if strings.EqualFold(surface, "ａｌｉｃｅ") {
		return "アリス"
	}
	return ""
}

func (r *EnglishReader) Compile(surface string) (UnknownKanaReading, error) {
	if pron := ConventionalEnglishPronunciation(surface); pron != "" {
		if r.Accent == nil {
			return UnknownKanaReading{}, fmt.Errorf("missing accent rules")
		}
		a, err := r.Accent.Accent(pron)
		if err != nil {
			return UnknownKanaReading{}, err
		}
		return UnknownKanaReading{Pronunciation: pron, Accent: a}, nil
	}
	return CompileUnknownEnglish(surface, r.Spelling, r.Transliteration, r.Accent, r.Fallback)
}
