package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/a-kaibu/aoikane/internal/frontend"
)

func run() error {
	path := flag.String("reference", "private/reading-context-reference.json", "native word-range fixtures")
	flag.Parse()
	b, err := os.ReadFile(*path)
	if err != nil {
		return err
	}
	var reference struct {
		Source string `json:"source_sha256"`
		Cases  []struct {
			Words     []frontend.LanguageWord    `json:"words"`
			Current   int                        `json:"current"`
			Condition frontend.LanguageCondition `json:"condition"`
			Scope     frontend.ReadingContext    `json:"scope"`
			Result    bool                       `json:"result"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(b, &reference); err != nil {
		return err
	}
	if reference.Source != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(reference.Cases) == 0 {
		return fmt.Errorf("invalid context reference")
	}
	for i, c := range reference.Cases {
		rules := frontend.ReadingRules{Conditions: map[int]frontend.ReadingPredicate{1: {Kind: "word", Condition: c.Condition}}}
		got, err := rules.EvaluateContext(c.Scope, c.Words, c.Current)
		if err != nil {
			return fmt.Errorf("case %d: %w", i, err)
		}
		if got != c.Result {
			return fmt.Errorf("case %d: got %v want %v (scope=%+v current=%d)", i, got, c.Result, c.Scope, c.Current)
		}
	}
	fmt.Printf("Verified %d word-range reading contexts against native output\n", len(reference.Cases))
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
