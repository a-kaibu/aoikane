package portable

import (
	"reflect"
	"testing"
)

// Compare structured mora/phone output, not accent punctuation: each voice
// retains its own prosody while saying the same words as the kana reference.
func TestReportedReadings(t *testing.T) {
	e, err := Open(Options{AssetsDir: "../../assets"})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	cases := [][2]string{
		{"alice", "アリス"}, {"Alice", "アリス"}, {"ALICE", "アリス"},
		{"ａｌｉｃｅ", "アリス"}, {"Ａｌｉｃｅ", "アリス"},
		{"Aliceです。", "アリスです。"},
		{"Aliceはりんごを3つくださいと言った。", "アリスはりんごをミッツくださいと言った。"},
		{"1つ", "ヒトツ"}, {"2つ", "フタツ"}, {"3つ", "ミッツ"},
		{"4つ", "ヨッツ"}, {"5つ", "イツツ"}, {"6つ", "ムッツ"},
		{"7つ", "ナナツ"}, {"8つ", "ヤッツ"}, {"9つ", "ココノツ"},
		{"３つ", "ミッツ"}, {"三つ", "ミッツ"}, {"3つ目", "ミッツ目"},
		{"あと3つ", "あとミッツ"}, {"3つとか", "ミッツとか"},
		{"3つの方法", "ミッツの方法"}, {"3つずつ", "ミッツずつ"},
		{"りんごを3つください。", "りんごをミッツください。"},
	}
	for _, voice := range []string{VoiceAoi, VoiceAkane} {
		for _, tc := range cases {
			t.Run(voice+"/"+tc[0], func(t *testing.T) {
				phones := func(text string) []int {
					t.Helper()
					kana, err := e.ReadingVoice(text, voice)
					if err != nil {
						t.Fatal(err)
					}
					var ids []int
					for _, k := range kana {
						s, err := e.encoder.ParseFormattedKana(k)
						if err != nil {
							t.Fatal(err)
						}
						if _, err := e.encoder.CompilePhones(s, 0xef); err != nil {
							t.Fatal(err)
						}
						for _, item := range s.Timeline {
							if item.Type != "mora" {
								continue
							}
							ids = append(ids, item.PhoneIndices...)
						}
					}
					return ids
				}
				got, want := phones(tc[0]), phones(tc[1])
				if len(got) == 0 || !reflect.DeepEqual(got, want) {
					t.Errorf("%q pronunciation differs from %q: got %v; want %v", tc[0], tc[1], got, want)
				}
			})
		}
	}
}
