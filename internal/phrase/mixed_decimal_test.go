package phrase

import (
	"github.com/a-kaibu/aoikane/internal/cabocha"
	"path/filepath"
	"testing"
)

// Captured from ordinary original text calls, including mixed scripts on both
// sides of the point and counters after the final run.
func TestMixedDecimalReadingsOriginal(t *testing.T) {
	f := openFrontend(t)
	var err error
	f.Dependency, err = cabocha.LoadSVM(filepath.Join("..", "..", "assets", "cabocha", "standard-dep.ipa.model"))
	if err != nil {
		t.Fatal(err)
	}
	f.Bound = openBoundRules(t, "bound-rules-standard.txt")
	f.Predictor = openPredictor(t)
	for _, tc := range []struct{ text, want string }{
		{"1.二。", "<S>^イ!チテン|0ニ<F>"},
		{"二.1。", "<S>^ニ!テン|0イ^チ<F>"},
		{"一二.３四。", "<S>ジュ^ーニ!テン|0サ^ン|0^ヨ!ン<F>"},
		{"私は1.2と三.4を見る。", "<S>ワ^タシワ$1_1^イ!ッテン|0^ニ!ト|0^サ!ンテン|0^ヨ!ンオ|0^ミ!ル<F>"},
		{"1.二三。", "<S>^イ!チテン|0^ニ!ジュー|0サ^ン<F>"},
		{"一.２３。", "<S>^イ!チテン|0^ニ!ジュー|0サ^ン<F>"},
		{"一二.３４。", "<S>ジュ^ーニ!テン|0^サ!ンジュー|0^ヨ!ン<F>"},
		{"1.2三4。", "<S>^イ!チテン|0ニ|0サ^ン|0^ヨ!ン<F>"},
		{"1.二3四。", "<S>^イ!チテン|0ニ|0サ^ン|0^ヨ!ン<F>"},
		{"一2.3。", "<S>イ^チ|0^ニ!テン|0サ^ン<F>"},
		{"一2.三。", "<S>イ^チ|0^ニ!テン|0サ^ン<F>"},
		{"1.二三人。", "<S>^イ!チテン|0^ニ!ジュー|0サ^ンニ!ン<F>"},
		{"二.12。", "<S>^ニ!テン|0ジュ^ーニ<F>"},
		{"12.三四。", "<S>ジュ^ーニ!テン|0^サ!ンジュー|0^ヨ!ン<F>"},

		{"0.一。", "<S>^ゼ!ロテン|0イ^チ<F>"},
		{"八.8。", "<S>^ハ!チテン|0ハ^チ<F>"},
		{"100.二。", "<S>^ヒャ!クテン|0ニ<F>"},
		{"一二三.45。", "<S>ヒャ^ク|0^ニ!ジュー|0^サ!ンテン|0^ヨ!ンジュー|0ゴ<F>"},
		{"12.三四人。", "<S>ジュ^ーニ!テン|0^サ!ンジュー|0ヨ^ニ!ン<F>"},
		{"1.二万。", "<S>^イ!チテン|0ニ^マ!ン<F>"},
		{"二.1万。", "<S>^ニ!テン|0イ^チマ!ン<F>"},
		{"5.五。", "<S>^ゴ!テン|0ゴ<F>"},
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
