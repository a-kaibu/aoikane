package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/a-kaibu/aoikane/internal/frontend"
)

func run() error {
	path := flag.String("reference", "private/reading-whole-reference.json", "native whole-context fixtures")
	flag.Parse()
	b, err := os.ReadFile(*path)
	if err != nil {
		return err
	}
	var reference struct {
		Source string `json:"source_sha256"`
		Cases  []struct {
			Words     []frontend.LanguageWord    `json:"words"`
			Text      string                     `json:"text"`
			All       bool                       `json:"all"`
			Condition frontend.LanguageCondition `json:"condition"`
			Result    bool                       `json:"result"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(b, &reference); err != nil {
		return err
	}
	if reference.Source != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(reference.Cases) == 0 {
		return fmt.Errorf("invalid whole-context reference")
	}
	for i, c := range reference.Cases {
		rules := frontend.ReadingRules{Conditions: map[int]frontend.ReadingPredicate{
			1: {Kind: "word", Condition: c.Condition},
			2: {Kind: "and"},
		}}
		got, err := rules.EvaluateWholeCondition(1, c.Words, c.Text, c.All)
		if err != nil {
			return fmt.Errorf("case %d: %w", i, err)
		}
		if got != c.Result {
			return fmt.Errorf("case %d: got %v want %v (%+v / %q)", i, got, c.Result, c.Condition, c.Text)
		}
		if len(c.Words) > 0 {
			rules.Rules = []frontend.ReadingRule{{Surface: c.Words[0].Surface, Reading: "second", Context: []frontend.ReadingContext{
				{Direction: "C", Condition: 2},
				{Direction: "N", Begin: 0, End: -1, All: c.All, Condition: 1},
			}}}
			selection, err := rules.SelectReadingWithContextText(c.Words, 0, []string{"first", "second"}, c.Text)
			if err != nil {
				return fmt.Errorf("case %d selector: %w", i, err)
			}
			want := frontend.ReadingSelection{Rule: -1}
			if c.Result {
				want = frontend.ReadingSelection{Rule: 0, Variant: 1}
			}
			if selection != want {
				return fmt.Errorf("case %d selector: got %+v want %+v", i, selection, want)
			}
		}
	}
	fmt.Printf("Verified %d whole-context reading predicates against native output\n", len(reference.Cases))
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
