package phrase

import (
	"github.com/a-kaibu/aoikane/internal/cabocha"
	"path/filepath"
	"testing"
)

// These readings come from normal original text calls. They remain available
// to regression tests without the private stage capture files.
func TestDecimalCounterReadingsOriginal(t *testing.T) {
	f := openFrontend(t)
	var err error
	f.Dependency, err = cabocha.LoadSVM(filepath.Join("..", "..", "assets", "cabocha", "standard-dep.ipa.model"))
	if err != nil {
		t.Fatal(err)
	}
	f.Bound = openBoundRules(t, "bound-rules-standard.txt")
	f.Predictor = openPredictor(t)
	for _, tc := range []struct{ text, want string }{
		{"1.23人。", "<S>^イ!ッテン|0ニ^ーサ!ンニン<F>"},
		{"1.24人。", "<S>^イ!ッテン|0ニ^ーヨ!ンニン<F>"},
		{"1.12個。", "<S>^イ!ッテン|0イ^チニ!ーコ<F>"},
		{"1.56本。", "<S>^イ!ッテン|0ゴ^ーロク!ホン<F>"},
		{"1.00円。", "<S>^イ!ッテン|0レ^ーレ!ーエン<F>"},
		{"1.1234人。", "<S>^イ!ッテン|0イ^チニ!ー|0サ^ンヨ!ンニン<F>"},
		{"1.23個。", "<S>^イ!ッテン|0ニ^ーサ!ンコ<F>"},
		{"1.23メートル。", "<S>^イ!ッテン|0ニ^ーサンメ!ートル<F>"},
		{"1.23番。", "<S>^イ!ッテン|0ニ^ーサ!ンバン<F>"},
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
