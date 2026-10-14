package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/a-kaibu/aoikane/internal/frontend"
)

func run() error {
	path := flag.String("reference", "private/language-condition-reference.json", "native predicate fixtures")
	rulesPath := flag.String("rules", "", "also load a UTF-8 reading rule table")
	flag.Parse()
	b, err := os.ReadFile(*path)
	if err != nil {
		return err
	}
	var reference struct {
		Source string `json:"source_sha256"`
		Cases  []struct {
			Condition frontend.LanguageCondition `json:"condition"`
			Word      frontend.LanguageWord      `json:"word"`
			Result    bool                       `json:"result"`
		} `json:"cases"`
		Compounds []struct {
			Conditions []frontend.LanguageCondition `json:"conditions"`
			Word       frontend.LanguageWord        `json:"word"`
			Negate     bool                         `json:"negate"`
			Result     bool                         `json:"result"`
		} `json:"compound_cases"`
	}
	if err := json.Unmarshal(b, &reference); err != nil {
		return err
	}
	if reference.Source != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(reference.Cases) == 0 {
		return fmt.Errorf("invalid predicate reference")
	}
	for i, c := range reference.Cases {
		got, err := c.Condition.Evaluate(c.Word)
		if err != nil {
			return fmt.Errorf("case %d: %w", i, err)
		}
		if got != c.Result {
			return fmt.Errorf("case %d: got %v want %v (%+v / %+v)", i, got, c.Result, c.Condition, c.Word)
		}
	}
	fmt.Printf("Verified %d language predicates against native output\n", len(reference.Cases))
	for i, c := range reference.Compounds {
		rules := frontend.ReadingRules{Conditions: make(map[int]frontend.ReadingPredicate)}
		group := frontend.ReadingPredicate{Kind: "and", Negate: c.Negate}
		for j, condition := range c.Conditions {
			rules.Conditions[j+1] = frontend.ReadingPredicate{Kind: "word", Condition: condition}
			group.Children = append(group.Children, j+1)
		}
		id := len(c.Conditions) + 1
		rules.Conditions[id] = group
		got, err := rules.EvaluateCondition(id, c.Word, false, false)
		if err != nil {
			return err
		}
		if got != c.Result {
			return fmt.Errorf("conjunction %d: got %v want %v", i, got, c.Result)
		}
	}
	fmt.Printf("Verified %d compound predicates against native output\n", len(reference.Compounds))
	if *rulesPath != "" {
		f, err := os.Open(*rulesPath)
		if err != nil {
			return err
		}
		defer f.Close()
		rules, err := frontend.ParseReadingRules(f)
		if err != nil {
			return err
		}
		fmt.Printf("Loaded %d reading rules and %d predicate definitions; all references resolve\n", len(rules.Rules), len(rules.Conditions))
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
