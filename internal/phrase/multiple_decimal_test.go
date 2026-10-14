package phrase

import (
	"github.com/a-kaibu/aoikane/internal/cabocha"
	"path/filepath"
	"testing"
)

// Expected readings come from ordinary original text calls. This exercises
// double decimal fractions and the general point-series connector separately.
func TestMultipleDecimalReadingsOriginal(t *testing.T) {
	f := openFrontend(t)
	var err error
	f.Dependency, err = cabocha.LoadSVM(filepath.Join("..", "..", "assets", "cabocha", "standard-dep.ipa.model"))
	if err != nil {
		t.Fatal(err)
	}
	f.Bound = openBoundRules(t, "bound-rules-standard.txt")
	f.Predictor = openPredictor(t)
	for _, tc := range []struct{ text, want string }{
		{"1.2.3。", "<S>^イ!ッテン|0^ニ!テン|0サ^ン<F>"},
		{"1.23.4。", "<S>^イ!ッテン|0^ニ!ーサンテン|0^ヨ!ン<F>"},
		{"12.34.56。", "<S>ジュ^ーニ!ーテン|0サ^ンヨ!ンテン|0ゴ^ーロ!ク<F>"},
		{"2.1.8。", "<S>^ニ!ーテン|0^イ!チテン|0ハ^チ<F>"},
		{"1.2.3人。", "<S>^イ!ッテン|0^ニ!テン|0^サ!ンニン<F>"},
		{"一.2.三。", "<S>^イ!チテン|0^ニ!テン|0サ^ン<F>"},
		{"1.2.3.4。", "<S>^イ!チテン|0^ニ!テン|0^サ!ンテン|0^ヨ!ン<F>"},
		{"1.23.45個。", "<S>^イ!ッテン|0^ニ!ーサンテン|0ヨ^ンゴ!ーコ<F>"},
		{"1.23.45.6。", "<S>^イ!チテン|0^ニ!ジュー|0^サ!ンテン|0^ヨ!ンジュー|0^ゴ!テン|0ロ^ク<F>"},
		{"1.23.4万。", "<S>^イ!チテン|0^ニ!ジュー|0^サ!ンテン|0ヨ^ンマ!ン<F>"},
		{"1.123.45。", "<S>^イ!ッテン|0イ^チニ!ー|0^サ!ンテン|0ヨ^ンゴ!ー<F>"},
		{"1.00.2。", "<S>^イ!ッテン|0^レ!ーレーテン|0ニ<F>"},
		{"1.2.34人。", "<S>^イ!ッテン|0^ニ!テン|0サ^ンヨ!ンニン<F>"},
		{"1.2.345。", "<S>^イ!ッテン|0^ニ!テン|0サ^ンヨ!ン|0ゴ<F>"},
		{"1.02.3。", "<S>^イ!ッテン|0^レ!ーニーテン|0サ^ン<F>"},
		{"1万.2.3。", "<S>イ^チマ!ン|0テ^ン|0^ニ!テン|0サ^ン<F>"},
		{"1.2万.3。", "<S>^イ!チテン|0ニ^マ!ン|0テ^ン|0サ^ン<F>"},
		{"一.二.三。", "<S>^イ!チテン|0^ニ!テン|0サ^ン<F>"},
	} {
		t.Run(tc.text, func(t *testing.T) {
			got, err := f.Kana(tc.text)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("reading=%s original=%s", got, tc.want)
			}
		})
	}
}
