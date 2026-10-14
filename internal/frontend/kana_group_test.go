package frontend

import (
	"reflect"
	"testing"
)

func TestSelectKanaRunsBoundaries(t *testing.T) {
	word := func(s string, origin int) KanaGroupWord {
		return KanaGroupWord{KanaMergeWord: KanaMergeWord{Surface: s}, OriginCount: origin}
	}
	for _, tc := range []struct {
		name  string
		words []KanaGroupWord
		want  []KanaRun
	}{
		{"empty", nil, nil},
		{"single unknown", []KanaGroupWord{word("ヌァ", 0)}, []KanaRun{{0, 1}}},
		{"punctuation and Latin", []KanaGroupWord{word("カナ", 0), word("、", 0), word("カナ", 0), word("abc", 0), word("ー", 0)}, []KanaRun{{0, 1}, {2, 3}, {4, 5}}},
		{"origin splits before current", []KanaGroupWord{word("カ", 0), word("ナ", 1), word("カ", 0), word("ナ", 2)}, []KanaRun{{0, 1}, {1, 3}, {3, 4}}},
		{"first origin and empty surface", []KanaGroupWord{word("カ", 1), word("ナ", 0), word("", 0), word("かな", 0)}, []KanaRun{{0, 2}}},
		{"kana classification edges", []KanaGroupWord{word("ヵヶヴー", 0), word("゜", 0), word("ヷ", 0), word("ァ", 0)}, []KanaRun{{0, 1}, {3, 4}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := SelectKanaRuns(tc.words)
			if err != nil || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v, %v; want %v", got, err, tc.want)
			}
		})
	}
}
