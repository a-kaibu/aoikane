package phrase

import (
	"reflect"
	"testing"
)

// These sentence boundaries were observed in normal original-DLL frontend
// calls, including empty readings; they are not inferred from Go's output.
func TestOriginalSentenceBoundaries(t *testing.T) {
	for _, tc := range []struct {
		text string
		want []string
	}{
		{"\nあ。", []string{"", "あ。"}},
		{"\r\nあ。", []string{"", "あ。"}},
		{"\n\nあ。", []string{"", "あ。"}},
		{" \nあ。", []string{"", "あ。"}},
		{"\n？あ。", []string{"", "あ。"}},
		{"\n）！あ。", []string{"", "）！", "あ。"}},
		{"\n。", []string{""}},
		{"？。", []string{"？"}},
		{"！！。", []string{"！"}},
		{"。あ。", []string{"。", "あ。"}},
		{"あ。。", []string{"あ。"}},
		{"。\n。", []string{"。"}},
		{"。？。", []string{"。"}},
		{"−－。？é？。", []string{"−－。", "é？"}},
		{"😀０～\n\t\t。", []string{"😀０～", "。"}},
		{"！A。", []string{"！", "A。"}},
		{"？4。", []string{"？", "4。"}},
		{"あ。？。", []string{"あ。"}},
		{"あ。\n。", []string{"あ。"}},
		{"あ\n\nあ", []string{"あ", "あ"}},
		{" －「（\n？。", []string{"－「（"}},
		{"－\n？。", []string{"－"}},
		{"（\n？。", []string{"（"}},
		{"「\n？。", []string{"「"}},
		{"－（\n？。", []string{"－（"}},
		{"あ（\n？。", []string{"あ（"}},
		{"－\nあ。", []string{"－", "あ。"}},
		{"あ\n？。", []string{"あ"}},
		{"あ\n。", []string{"あ"}},
		{"－\n。", []string{"－"}},
		{"－\n！", []string{"－"}},
		{"－\n?", []string{"－"}},
		{"あ\n？", []string{"あ"}},
		{"あ\n！", []string{"あ"}},
		{"－\n\t。", []string{"－", "。"}},
		{"－\n 。", []string{"－", "。"}},
		{"－\n　。", []string{"－", "。"}},
		{"－\n\n。", []string{"－"}},
		{"－\r。", []string{"－"}},
		{"－\n\t？。", []string{"－", "？"}},
		{"😀０～\n。", []string{"😀０～"}},
	} {
		t.Run(tc.text, func(t *testing.T) {
			if got := SplitSentences(tc.text); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("sentences=%q want=%q", got, tc.want)
			}
		})
	}
}
