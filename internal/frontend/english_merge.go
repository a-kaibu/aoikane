package frontend

import "fmt"

// EnglishReadingFunc runs the relevant branch of 1a4490. An unresolved branch
// must return an error rather than substitute a different pronunciation.
type EnglishReadingFunc func(surface string) (UnknownKanaReading, error)

// MergeEnglishRun reconstructs 1a4fd0 for a caller-selected alphabetic run.
// It uses asymmetric short-word limits (4/2 CP932 bytes), unlike kana merging.
func MergeEnglishRun(words []KanaMergeWord, codec *CP932Codec, read EnglishReadingFunc) ([]KanaMergeWord, error) {
	if len(words) == 0 || len(words) > 8192 || codec == nil || read == nil {
		return nil, fmt.Errorf("invalid English merge input")
	}
	result := append([]KanaMergeWord(nil), words...)
	lengths := make([]int, len(words))
	for i, word := range words {
		if word.Surface == "" {
			return nil, fmt.Errorf("empty English merge surface")
		}
		for _, c := range word.Surface {
			if !(c >= 'Ａ' && c <= 'Ｚ' || c >= 'ａ' && c <= 'ｚ') {
				return nil, fmt.Errorf("non-alphabetic English merge surface")
			}
		}
		b, err := codec.Encode(word.Surface)
		if err != nil {
			return nil, err
		}
		lengths[i] = len(b)
	}
	for i := 0; i < len(result); i++ {
		for i+1 < len(result) && lengths[i]+lengths[i+1] < 60 &&
			(result[i].POS == 0x0d030000 || result[i+1].POS == 0x0d030000 || lengths[i] <= 4 || lengths[i+1] <= 2) {
			result[i].Surface += result[i+1].Surface
			result[i].Base = result[i].Surface
			result[i].POS = 0x0d030000
			lengths[i] += lengths[i+1]
			result = append(result[:i+1], result[i+2:]...)
			lengths = append(lengths[:i+1], lengths[i+2:]...)
		}
		if result[i].POS == 0x0d030000 {
			reading, err := read(result[i].Surface)
			if err != nil {
				return nil, err
			}
			result[i].Pronunciation = reading.Pronunciation
			result[i].Generated = &reading
			result[i].POS = 0x01010400
		}
	}
	return result, nil
}
