// Compare native unknown candidate creation order and metadata, including
// positions where a dictionary match suppresses unknown generation.
package main

import (
	"github.com/a-kaibu/aoikane/internal/frontend"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

type event struct {
	Event      string                 `json:"event"`
	Text       string                 `json:"text"`
	Info       frontend.CharacterInfo `json:"info"`
	Begin      int                    `json:"begin"`
	End        int                    `json:"end"`
	Matches    []json.RawMessage      `json:"matches"`
	Candidates []struct {
		Context       uint16 `json:"context"`
		WordCost      int16  `json:"word_cost"`
		FeatureOffset uint32 `json:"feature_offset"`
	} `json:"candidates"`
}

func run() error {
	dir := flag.String("dictionary", "private/frontend-data", "SMecab dictionary directory")
	trace := flag.String("trace", "private/unknown-character-trace", "native lexical trace directory")
	maxGroup := flag.Int("max-group", 24, "native caller's maximum grouped character count")
	flag.Parse()
	c, err := frontend.OpenCharacters(filepath.Join(*dir, "char.bin"))
	if err != nil {
		return err
	}
	b, err := os.ReadFile(filepath.Join(*trace, "events.json"))
	if err != nil {
		return err
	}
	var events []event
	if err = json.Unmarshal(b, &events); err != nil {
		return err
	}
	positions, candidates := 0, 0
	var expected []frontend.UnknownCandidate
	at := 0
	havePrefix := false
	finish := func() error {
		if havePrefix && at != len(expected) {
			return fmt.Errorf("missing native unknown calls: Go=%d native=%d", len(expected), at)
		}
		return nil
	}
	for _, e := range events {
		switch e.Event {
		case "dictionary_prefix":
			if err = finish(); err != nil {
				return err
			}
			expected, err = c.UnknownCandidates(e.Text, len(e.Matches) > 0, *maxGroup)
			if err != nil {
				return err
			}
			positions++
			at = 0
			havePrefix = true
		case "unknown_candidates":
			if !havePrefix || at >= len(expected) {
				return fmt.Errorf("unexpected native unknown candidate %q", e.Text)
			}
			got := expected[at]
			if got.Text != e.Text || got.Info != e.Info || got.Characters != e.End-e.Begin || len(e.Candidates) != 1 {
				return fmt.Errorf("candidate mismatch for %q at %d: Go=%+v native=%+v", e.Text, at, got, e)
			}
			n := e.Candidates[0]
			if n.Context != got.Context || n.WordCost != got.WordCost || n.FeatureOffset != 0 {
				return fmt.Errorf("unknown metadata mismatch for %q", e.Text)
			}
			at++
			candidates++
		}
	}
	if err = finish(); err != nil {
		return err
	}
	if positions == 0 || candidates == 0 {
		return fmt.Errorf("trace has no unknown candidates")
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]int{"positions": positions, "candidates": candidates})
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
