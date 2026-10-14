package phrase

import (
	"github.com/a-kaibu/aoikane/internal/cabocha"
	"github.com/a-kaibu/aoikane/internal/frontend"
	"os"
	"path/filepath"
	"testing"
)

func TestReadingChunksTextOriginal(t *testing.T) {
	for _, language := range []string{"standard", "kansai"} {
		t.Run(language, func(t *testing.T) {
			f := openFrontend(t)
			var err error
			f.Dependency, err = cabocha.LoadSVM(filepath.Join("..", "..", "private", "cabocha", "standard-dep.ipa.model"))
			if os.IsNotExist(err) {
				t.Skip("private dependency model unavailable")
			}
			if err != nil {
				t.Fatal(err)
			}
			folder := "stages_aoi"
			if language == "kansai" {
				folder = "stages_akane"
				ruleFile, ruleErr := os.Open(filepath.Join("..", "..", "private", "reading-rules-kansai-utf8.txt"))
				if os.IsNotExist(ruleErr) {
					t.Skip("private Kansai reading rules unavailable")
				}
				if ruleErr != nil {
					t.Fatal(ruleErr)
				}
				defer ruleFile.Close()
				f.Reading, err = frontend.ParseReadingRules(ruleFile)
				if err != nil {
					t.Fatal(err)
				}
				f.Analyzer, err = frontend.OpenAnalyzer(filepath.Join("..", "..", "private", "frontend-reconstruction", "frontend-data", "kansai", "lang", "smecab"))
				if os.IsNotExist(err) {
					t.Skip("private Kansai dictionary unavailable")
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			cases := aoiCases(t, folder)
			if len(cases) == 0 {
				t.Skip("private original graph references unavailable")
			}
			if len(cases) != 79 {
				t.Fatalf("incomplete original graph references: %d", len(cases))
			}
			for _, c := range cases {
				want, err := c.Leave.Load(f.Codec)
				if err != nil {
					t.Fatal(err)
				}
				// These captures stop after the first frontend call. The native
				// caller splits multi-sentence input before invoking that frontend.
				text := SplitSentences(c.Text)[0]
				got, err := f.Graph(text)
				if err != nil {
					t.Fatalf("%q: %v", c.Text, err)
				}
				if diff := compareGraphs(got, want); diff != "" {
					t.Fatalf("%q: %s", c.Text, diff)
				}
			}
			t.Logf("%d original %s texts -> Go word graphs: identical, no skipped phrase scopes", len(cases), language)
		})
	}
}
