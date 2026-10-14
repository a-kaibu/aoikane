package frontend

import "testing"

func TestEnglishReaderNativeReadings(t *testing.T) {
	codec, err := OpenCP932Codec("../../assets/data/cp932.json")
	if err != nil {
		t.Fatal(err)
	}
	accent, err := OpenUnknownAccentRules("../../assets/rules/unknown-accent.json")
	if err != nil {
		t.Fatal(err)
	}
	r, err := OpenEnglishReader("../../assets/rules", codec, accent)
	if err != nil {
		t.Fatal(err)
	}
	// Exact pronunciation/accent values from the unmodified engine's
	// 1ba850 output in the unknown-English corpus captured on 2026-10-08.
	for _, tc := range []struct {
		surface, pronunciation string
		moras, down            int
	}{
		{"Ｇｐｔ", "ジーピーティー", 6, 5},
		{"ｓｕｓｈｉ", "スシ", 2, 1},
		{"ｃｌｙｒｐ", "クリルプ", 4, 0},
		{"ｆｌｏｒｐ", "フロルプ", 4, 0},
		{"ｔｈｒｏｓｈ", "スロッシュ", 4, 2},
		{"ｓｎａｒｆ", "スナールフ", 5, 2},
	} {
		t.Run(tc.surface, func(t *testing.T) {
			got, err := r.Compile(tc.surface)
			if err != nil {
				t.Fatal(err)
			}
			if got.Pronunciation != tc.pronunciation || got.Accent.Moras != tc.moras || got.Accent.Down != tc.down {
				t.Fatalf("got %+v; native %s, %d moras, fall %d", got, tc.pronunciation, tc.moras, tc.down)
			}
		})
	}
}
