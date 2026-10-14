package frontend

import "fmt"

type KanaMergeWord struct {
	Surface       string              `json:"surface"`
	Base          string              `json:"base"`
	POS           uint32              `json:"pos"`
	Pronunciation string              `json:"pronunciation"`
	Generated     *UnknownKanaReading `json:"generated,omitempty"`
}

// MergeKanaRun reconstructs 1a52c0 on a caller-selected, contiguous kana run.
// Selecting runs, sentence/group boundaries, and source-origin split points is
// a preceding pass; this function does not infer those from the flattened text.
func MergeKanaRun(words []KanaMergeWord, codec *CP932Codec, kana *UnknownKanaRules, accent *UnknownAccentRules) ([]KanaMergeWord, error) {
	if len(words) > 8192 || codec == nil || kana == nil || accent == nil {
		return nil, fmt.Errorf("invalid kana merge input")
	}
	result := append([]KanaMergeWord(nil), words...)
	lengths := make([]int, len(result))
	for i, w := range result {
		if w.Surface == "" {
			return nil, fmt.Errorf("empty kana merge surface")
		}
		for _, c := range w.Surface {
			if !(c >= 0x30a1 && c <= 0x30f6 || c == 0x30fc) {
				return nil, fmt.Errorf("non-katakana in kana merge run")
			}
		}
		b, err := codec.Encode(w.Surface)
		if err != nil {
			return nil, err
		}
		lengths[i] = len(b)
	}
	begin := 0
	for begin < len(result) && (result[begin].Pronunciation == "ン" || result[begin].Pronunciation == "ッ" || result[begin].Pronunciation == "ー") {
		begin++
	}
	finish := func(i int) error {
		if result[i].POS != 0x0d030000 {
			return nil
		}
		reading, err := CompileUnknownKana(result[i].Surface, kana, accent)
		if err != nil {
			return err
		}
		result[i].Base = result[i].Surface
		result[i].Pronunciation = reading.Pronunciation
		result[i].Generated = &reading
		result[i].POS = 0x01040000
		return nil
	}
	for i := begin; i < len(result); i++ {
		for i+1 < len(result) {
			canMerge := lengths[i]+lengths[i+1] < 60 &&
				(result[i].POS == 0x0d030000 || result[i+1].POS == 0x0d030000 || lengths[i] <= 4 || lengths[i+1] <= 4)
			if !canMerge {
				break
			}
			result[i].Surface += result[i+1].Surface
			result[i].Base = result[i].Surface
			result[i].POS = 0x0d030000
			lengths[i] += lengths[i+1]
			result = append(result[:i+1], result[i+2:]...)
			lengths = append(lengths[:i+1], lengths[i+2:]...)
		}
		if err := finish(i); err != nil {
			return nil, err
		}
	}
	return result, nil
}
