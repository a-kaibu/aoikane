package phrase

import (
	"github.com/a-kaibu/aoikane/internal/cabocha"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Expected readings were captured from normal original-engine calls, not from
// the reconstructed numeral reader. Decimal coverage is in mixed_decimal_test.go.
func TestMixedDigitReadingsOriginal(t *testing.T) {
	f := openFrontend(t)
	var err error
	f.Dependency, err = cabocha.LoadSVM(filepath.Join("..", "..", "assets", "cabocha", "standard-dep.ipa.model"))
	if err != nil {
		t.Fatal(err)
	}
	f.Bound = openBoundRules(t, "bound-rules-standard.txt")
	f.Predictor = openPredictor(t)
	for _, tc := range []struct{ text, want string }{
		{"声八一7。", "<S>^コ!エ|0ハ^チジュー|0イ^チ|0^ナ!ナ<F>"},
		{"水土Ａ7二2。", "<S>ス^イドエ!ー|0^ナ!ナ|0ニ|0ニ<F>"},
		{"八一7。", "<S>ハ^チジュー|0イ^チ|0^ナ!ナ<F>"},
		{"7二2。", "<S>^ナ!ナ|0ニ|0ニ<F>"},
		{"一2。", "<S>イ^チ|0ニ<F>"},
		{"1二。", "<S>イ^チ|0ニ<F>"},
		{"1万2。", "<S>イ^チマ!ン|0ニ<F>"},
		{"一万2。", "<S>イ^チマ!ン|0ニ<F>"},
		{"1万二。", "<S>イ^チマ!ン|0ニ<F>"},
		{"123四567。", "<S>ヒャ^ク|0^ニ!ジュー|0サ^ン|0^ヨ!ン|0ゴ^ヒャク|0ロ^クジュー|0^ナ!ナ<F>"},
		{"七２三。", "<S>^ナ!ナ|0ニ|0サ^ン<F>"},
		{"八一７人。", "<S>ハ^チジュー|0イ^チ|0シ^チ!ニン<F>"},
		{"7二2人。", "<S>^ナ!ナ|0ニ|0^ニ!ニン<F>"},
		{"1億二千万3。", "<S>イ^チ!オク|0ニ^センマ!ン|0サ^ン<F>"},
		{"一２三四。", "<S>イ^チ|0ニ|0^サ!ンジュー|0^ヨ!ン<F>"},
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

func TestMixedNumeralGraphOriginal(t *testing.T) {
	f := openFrontend(t)
	var err error
	f.Dependency, err = cabocha.LoadSVM(filepath.Join("..", "..", "assets", "cabocha", "standard-dep.ipa.model"))
	if err != nil {
		t.Fatal(err)
	}
	hashes := []string{"f9f43219ad238aa6e5026e882fd2c1b1f8c89206eb2142dacff1bfd5fe62e8fc", "d504eb02444cddea4a601a587756f7dd03ce555aa0e15f2777d8a8608d5dba7a", "d24d607a3de698fcefdd522a6bf6c11d77f8de0fab4704547842780bccefb507"}
	for i := 0; i < 3; i++ {
		path := fmt.Sprintf("../../private/stages_randomnum/c%d.jsonl", i)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Skip("fresh original mixed numeral graph unavailable")
		} else if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if fmt.Sprintf("%x", sha256.Sum256(data)) != hashes[i] {
			t.Fatal("original numeral graph reference differs")
		}
		snapshots, err := ReadSnapshots(path)
		if err != nil {
			t.Fatal(err)
		}
		compared := false
		for _, s := range snapshots {
			if s.Event != "stage_leave_1b9f10" {
				continue
			}
			compared = true
			t.Run(s.Input, func(t *testing.T) {
				want, err := s.Load(f.Codec)
				if err != nil {
					t.Fatal(err)
				}
				got, err := f.Graph(s.Input)
				if err != nil {
					t.Fatal(err)
				}
				if diff := compareGraphs(got, want); diff != "" {
					t.Fatal(diff)
				}
			})
		}
		if !compared {
			t.Fatal("missing original BANCHA result")
		}
	}
}
