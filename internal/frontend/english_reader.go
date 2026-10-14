package frontend

import "path/filepath"

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

func (r *EnglishReader) Compile(surface string) (UnknownKanaReading, error) {
	return CompileUnknownEnglish(surface, r.Spelling, r.Transliteration, r.Accent, r.Fallback)
}
