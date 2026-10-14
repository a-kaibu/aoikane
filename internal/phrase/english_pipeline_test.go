package phrase

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/a-kaibu/aoikane/internal/frontend"
	"github.com/a-kaibu/aoikane/internal/synthesis"
)

func TestUnknownEnglishMatchesNative(t *testing.T) {
	testNativeTextCorpus(t, "stages_english_unknown")
}

func TestLongMoraTextMatchesNative(t *testing.T) {
	testNativeTextCorpus(t, "stages_long_mora")
}

func testNativeTextCorpus(t *testing.T, dir string) {
	t.Helper()
	codec := testCodec(t)
	fe := openFrontend(t)
	fe.Bound = openBoundRules(t, "bound-rules-standard.txt")
	fe.Predictor = openPredictor(t)
	fe.Reading.SkipPhraseScopes = true
	cases := aoiCases(t, dir)
	if len(cases) == 0 {
		t.Skip("native text corpus is not available")
	}
	for _, c := range cases {
		t.Run(c.Text, func(t *testing.T) {
			want, err := c.Leave.Load(codec)
			if err != nil {
				t.Fatal(err)
			}
			got, err := fe.Graph(c.Text)
			if err != nil {
				t.Fatal(err)
			}
			if diff := compareGraphs(got, want); diff != "" {
				t.Error(diff)
			}
			wantText, err := c.Done.FormatText(codec)
			if err != nil {
				t.Fatal(err)
			}
			gotText, err := fe.Kana(c.Text)
			if err != nil {
				t.Fatal(err)
			}
			if gotText != wantText {
				t.Errorf("got %s; native %s", gotText, wantText)
			}
		})
	}
	t.Logf("compared %d native graphs and formatted readings", len(cases))
}

func TestUnknownEnglishNativeSynthesisInput(t *testing.T) {
	testNativeSynthesisInputs(t, "../../private/english-native-input.json", "florpとthroshとsnarfを入力した。")
}

func TestLongMoraNativeSynthesisInputs(t *testing.T) {
	testNativeSynthesisInputs(t, "../../private/long-mora-native-input.json", "ーカナです。", "ンーカナです。", "ッーカナです。", "アーです。", "アーーです。")
}

func testNativeSynthesisInputs(t *testing.T, path string, texts ...string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		t.Skip("native synthesis input capture is not available")
	}
	if err != nil {
		t.Fatal(err)
	}
	var want []synthesis.Input
	if err := json.Unmarshal(b, &want); err != nil {
		t.Fatal(err)
	}
	if len(want) != len(texts) {
		t.Fatalf("expected %d native sentences, got %d", len(texts), len(want))
	}
	fe := openFrontend(t)
	fe.Bound = openBoundRules(t, "bound-rules-standard.txt")
	fe.Predictor = openPredictor(t)
	fe.Reading.SkipPhraseScopes = true
	e, err := frontend.OpenMoraEncoder("../../assets/data/phonetics.json")
	if err != nil {
		t.Fatal(err)
	}
	b, err = os.ReadFile("../../assets/data/controls-aoi.json")
	if err != nil {
		t.Fatal(err)
	}
	var controls synthesis.Controls
	if err := json.Unmarshal(b, &controls); err != nil {
		t.Fatal(err)
	}
	for i, text := range texts {
		t.Run(text, func(t *testing.T) {
			kana, err := fe.Kana(text)
			if err != nil {
				t.Fatal(err)
			}
			s, err := e.ParseFormattedKana(kana)
			if err != nil {
				t.Fatal(err)
			}
			phones, err := e.CompilePhones(s, 0xef)
			if err != nil {
				t.Fatal(err)
			}
			got, err := synthesis.FromPhones(phones, controls)
			if err != nil {
				t.Fatal(err)
			}
			// Both inputs decode as float32, including all model features,
			// style, controls and pauses. Native readings are never inputs.
			if !reflect.DeepEqual(got, want[i]) {
				t.Fatal("text-generated synthesis input differs from the native capture")
			}
		})
	}
}
