package frontend

import "fmt"

// MergeKanaEnglishGroup connects the kana pass and repeated English scans of
// 1a4770, up to (but excluding) its later word cleanup pass. Source-origin
// context transitions require a separate document-graph pass; reject them
// rather than infer a boundary from flattened words.
func MergeKanaEnglishGroup(words []KanaGroupWord, codec *CP932Codec, kana *UnknownKanaRules, accent *UnknownAccentRules, read EnglishReadingFunc) ([]KanaMergeWord, error) {
	if codec == nil || kana == nil || accent == nil || read == nil {
		return nil, fmt.Errorf("missing word-group reading rules")
	}
	for _, w := range words {
		if w.OriginCount != 0 {
			return nil, fmt.Errorf("word group requires source-origin context processing")
		}
	}
	kanaWords, err := MergeKanaGroup(words, codec, kana, accent)
	if err != nil {
		return nil, err
	}
	input := make([]EnglishGroupWord, len(kanaWords))
	for i, w := range kanaWords {
		input[i].KanaMergeWord = w
	}
	out := make([]KanaMergeWord, 0, len(input))
	for len(input) > 0 {
		merged, consumed, err := MergeEnglishGroupScan(input, codec, read)
		if err != nil {
			return nil, err
		}
		if consumed <= 0 || consumed > len(input) {
			return nil, fmt.Errorf("English group scan made no progress")
		}
		out = append(out, merged...)
		input = input[consumed:]
	}
	return out, nil
}
