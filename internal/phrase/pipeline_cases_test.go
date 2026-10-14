package phrase

import (
	"github.com/a-kaibu/aoikane/internal/cabocha"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// stageCase is one captured frontend call of the standard voice: the text of
// its sentence and the engine's snapshots around the BANCHA stage and of the
// formatted result.
type stageCase struct {
	Name, Text         string
	Enter, Leave, Done *Snapshot
}

// aoiCases reads the captured standard-voice stage snapshots. Single-sentence
// captures (c*.jsonl) carry their text; batch captures (b*.jsonl) hold one
// line of the sibling .txt per sentence.
func aoiCases(t *testing.T, dirs ...string) []stageCase {
	t.Helper()
	var out []stageCase
	for _, dir := range dirs {
		files, _ := filepath.Glob(filepath.Join("..", "..", "private", dir, "[cb]*.jsonl"))
		sort.Strings(files)
		for _, path := range files {
			snaps, err := ReadSnapshots(path)
			if err != nil {
				t.Fatal(err)
			}
			var lines []string
			if strings.HasPrefix(filepath.Base(path), "b") {
				b, err := os.ReadFile(strings.TrimSuffix(path, ".jsonl") + ".txt")
				if err != nil {
					continue
				}
				lines = SplitSentences(strings.NewReplacer("\r", "", "\n", "").Replace(string(b)))
			}
			cases := map[int]*stageCase{}
			for i := range snaps {
				s := &snaps[i]
				c := cases[s.Sentence]
				if c == nil {
					c = &stageCase{Name: filepath.Base(path), Text: s.Input}
					if lines != nil {
						if s.Sentence >= len(lines) {
							continue
						}
						c.Text = lines[s.Sentence]
					}
					cases[s.Sentence] = c
				}
				switch s.Event {
				case "stage_enter_1b9f10":
					c.Enter = s
				case "stage_leave_1b9f10":
					c.Leave = s
				case "format":
					c.Done = s
				}
			}
			var keys []int
			for k := range cases {
				keys = append(keys, k)
			}
			sort.Ints(keys)
			for _, k := range keys {
				if c := cases[k]; c.Enter != nil && c.Leave != nil && c.Done != nil {
					out = append(out, *c)
				}
			}
		}
	}
	return out
}

// TestGraphMatchesNative compares the analysed word graph with the engine's
// graph after its BANCHA reading selection.
func TestGraphMatchesNative(t *testing.T) {
	codec := testCodec(t)
	fe := openFrontend(t)
	var err error
	fe.Dependency, err = cabocha.LoadSVM(filepath.Join("..", "..", "assets", "cabocha", "standard-dep.ipa.model"))
	if err != nil {
		t.Fatal(err)
	}
	total, bad, skipped, numerals := 0, 0, 0, 0
	for _, c := range aoiCases(t, "stages_aoi", "stages_aoi3", "stages_aoi4") {
		want, err := c.Leave.Load(codec)
		if err != nil {
			t.Errorf("%s: %v", c.Name, err)
			continue
		}
		if multiSentence(c.Text) {
			skipped++
			continue
		}
		got, err := fe.Graph(c.Text)
		if err != nil {
			t.Errorf("%s: %v", c.Name, err)
			continue
		}
		total++
		if len(got.Morphs) != len(want.Morphs) {
			// the numeral pass (11f9c0) merges digit words, which is not
			// reconstructed yet
			numerals++
			t.Logf("%s: %s: morph count %d, native %d", c.Name, c.Text, len(got.Morphs), len(want.Morphs))
			continue
		}
		if diff := compareGraphs(got, want); diff != "" {
			bad++
			if bad <= 10 {
				t.Errorf("%s: %s\n%s", c.Name, c.Text, diff)
			}
		}
	}
	t.Logf("%d sentences, %d mismatches, %d with merged numerals, %d skipped", total, bad, numerals, skipped)
	if total != 169 || skipped != 1 || numerals != 0 {
		t.Errorf("incomplete graph comparison: %d compared, %d skipped, %d count differences", total, skipped, numerals)
	}
}

// TestEndToEndStandard runs the reconstructed frontend (analysis, boundary
// rules and predictor, phrase construction) and compares the formatted
// reading with the engine's.
func TestEndToEndStandard(t *testing.T) {
	codec := testCodec(t)
	fe := openFrontend(t)
	fe.Bound = openBoundRules(t, "bound-rules-standard.txt")
	fe.Predictor = openPredictor(t)
	var err error
	fe.Dependency, err = cabocha.LoadSVM(filepath.Join("..", "..", "assets", "cabocha", "standard-dep.ipa.model"))
	if err != nil {
		t.Fatal(err)
	}
	total, bad := 0, 0
	for _, c := range aoiCases(t, "stages_aoi", "stages_aoi3", "stages_aoi4") {
		if multiSentence(c.Text) {
			continue
		}
		want, err := c.Done.FormatText(codec)
		if err != nil {
			t.Fatalf("%s: native format: %v", c.Text, err)
		}
		got, err := fe.Kana(c.Text)
		if err != nil {
			t.Errorf("%s: %s: %v", c.Name, c.Text, err)
			continue
		}
		total++
		if got != want {
			bad++
			if bad <= 40 {
				t.Logf("%s: %s\n got  %s\n want %s", c.Name, c.Text, got, want)
			}
		}
	}
	t.Logf("%d sentences, %d differ", total, bad)
	if total != 169 {
		t.Fatalf("incomplete native standard sentence comparison: %d, want 169", total)
	}
	if bad > 0 {
		t.Errorf("%d formatted readings differ from the engine", bad)
	}
}

// TestEndToEndNumerals compares the formatted reading of the numeral corpora
// (integers, counters, decimals, patterns) with the engine's.
func TestEndToEndNumerals(t *testing.T) {
	codec := testCodec(t)
	fe := openFrontend(t)
	fe.Bound = openBoundRules(t, "bound-rules-standard.txt")
	fe.Predictor = openPredictor(t)
	fe.Reading.SkipPhraseScopes = true
	total, bad := 0, 0
	for _, c := range aoiCases(t, "stages_num", "stages_counter", "stages_dec", "stages_dec2", "stages_pat", "stages_pat2") {
		if multiSentence(c.Text) {
			continue
		}
		want, err := c.Done.FormatText(codec)
		if err != nil {
			t.Fatalf("%s: native format: %v", c.Text, err)
		}
		got, err := fe.Kana(c.Text)
		if err != nil {
			t.Errorf("%s: %v", c.Text, err)
			continue
		}
		total++
		if got != want {
			bad++
			t.Errorf("%s\n got  %s\n want %s", c.Text, got, want)
		}
	}
	t.Logf("%d sentences, %d differ", total, bad)
	if total == 0 {
		t.Fatal("no native numeral sentences were compared")
	}
}

// TestNumeralCorpus compares the analysed graph of the numeral corpora with the
// engine's after BANCHA (informational: patterns such as dates, times,
// decimals and phone numbers are not reconstructed).
func TestNumeralCorpus(t *testing.T) {
	codec := testCodec(t)
	fe := openFrontend(t)
	total, bad := 0, 0
	for _, c := range aoiCases(t, "stages_num", "stages_counter", "stages_dec", "stages_dec2", "stages_pat", "stages_pat2") {
		want, err := c.Leave.Load(codec)
		if err != nil || multiSentence(c.Text) {
			continue
		}
		got, err := fe.Graph(c.Text)
		if err != nil {
			continue
		}
		total++
		diff := compareGraphs(got, want)
		if diff != "" {
			bad++
			if bad <= 200 {
				t.Logf("%s: %s", c.Text, diff)
			}
		}
	}
	t.Logf("%d sentences, %d differ", total, bad)
}

// TestUnknownWords compares the graph of sentences with unknown words
// (katakana, emoji, rare kanji) with the engine's.
func TestUnknownWords(t *testing.T) {
	codec := testCodec(t)
	fe := openFrontend(t)
	for _, c := range aoiCases(t, "stages_unk") {
		want, err := c.Leave.Load(codec)
		if err != nil {
			continue
		}
		got, err := fe.Graph(c.Text)
		if err != nil {
			t.Logf("%s: %v", c.Text, err)
			continue
		}
		if diff := compareGraphs(got, want); diff != "" {
			t.Logf("%s: %s", c.Text, diff)
		}
		wantText, _ := c.Done.FormatText(codec)
		fe.Bound = openBoundRules(t, "bound-rules-standard.txt")
		fe.Predictor = openPredictor(t)
		if gotText, err := fe.Kana(c.Text); err == nil && gotText != wantText {
			t.Logf("%s: got %s want %s", c.Text, gotText, wantText)
		}
	}
}

// TestEnglishWords compares sentences with Latin text (informational: the
// kana/English word grouping of the engine is not connected yet).
func TestEnglishWords(t *testing.T) {
	codec := testCodec(t)
	fe := openFrontend(t)
	fe.Bound = openBoundRules(t, "bound-rules-standard.txt")
	fe.Predictor = openPredictor(t)
	fe.Reading.SkipPhraseScopes = true
	bad, total := 0, 0
	for _, c := range aoiCases(t, "stages_en") {
		want, _ := c.Done.FormatText(codec)
		got, err := fe.Kana(c.Text)
		total++
		if err != nil {
			bad++
			t.Logf("%s: %v", c.Text, err)
			continue
		}
		if got != want {
			bad++
			t.Logf("%s\n got  %s\n want %s", c.Text, got, want)
		}
	}
	t.Logf("%d sentences, %d differ", total, bad)
}
