package main

import (
	"github.com/a-kaibu/aoikane/internal/frontend"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func run() error {
	tables := flag.String("tables", "private/english-segments.json", "segmentation rules")
	reference := flag.String("reference", "private/english-context-reference.json", "native context references")
	trace := flag.String("trace", "", "optional normal-engine trace")
	flag.Parse()
	rules, err := frontend.OpenEnglishSegmentRules(*tables)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(*reference)
	if err != nil {
		return err
	}
	type contextCase struct {
		Text     *string                           `json:"text"`
		Prepared []frontend.PreparedEnglishSegment `json:"prepared"`
		Index    int                               `json:"index"`
		OptionA  int                               `json:"option_a"`
		OptionB  int                               `json:"option_b"`
		Context  frontend.EnglishConditionContext  `json:"context"`
	}
	var ref struct {
		Source string        `json:"source_sha256"`
		Cases  []contextCase `json:"cases"`
	}
	if err := json.Unmarshal(b, &ref); err != nil {
		return err
	}
	if ref.Source != rules.Source || len(ref.Cases) == 0 {
		return fmt.Errorf("invalid context reference")
	}
	check := func(c contextCase) error {
		units := c.Prepared
		if c.Text != nil {
			split, err := rules.Segment(*c.Text)
			if err != nil {
				return err
			}
			units, err = rules.Prepare(split)
			if err != nil {
				return err
			}
		}
		got, err := frontend.BuildEnglishConditionContext(units, c.Index, c.OptionA, c.OptionB)
		if err != nil {
			return err
		}
		if got != c.Context {
			return fmt.Errorf("context index %d got %+v want %+v", c.Index, got, c.Context)
		}
		return nil
	}
	for i, c := range ref.Cases {
		if err := check(c); err != nil {
			return fmt.Errorf("case %d: %w", i, err)
		}
	}
	fmt.Printf("Verified %d native English condition contexts\n", len(ref.Cases))
	if *trace == "" {
		return nil
	}
	b, err = os.ReadFile(*trace)
	if err != nil {
		return err
	}
	var events []json.RawMessage
	if err := json.Unmarshal(b, &events); err != nil {
		return err
	}
	count, models := 0, 0
	for _, raw := range events {
		var header struct {
			Event string `json:"event"`
		}
		if err := json.Unmarshal(raw, &header); err != nil {
			return err
		}
		switch header.Event {
		case "frontend_error", "forward_error":
			return fmt.Errorf("failed engine trace")
		case "torch_jit_result":
			models++
		case "frontend_english_context":
			var c contextCase
			if err := json.Unmarshal(raw, &c); err != nil {
				return err
			}
			if err := check(c); err != nil {
				return err
			}
			count++
		}
	}
	if count == 0 || models == 0 {
		return fmt.Errorf("incomplete normal context trace")
	}
	fmt.Printf("Verified %d normal-engine English condition contexts\n", count)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
