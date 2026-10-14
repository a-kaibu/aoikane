package main

import (
	"github.com/a-kaibu/aoikane/internal/frontend"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"reflect"
)

func run() error {
	tables := flag.String("tables", "private/english-decisions.json", "observed English trees")
	reference := flag.String("reference", "private/english-decisions-reference.json", "native evaluations")
	trace := flag.String("trace", "", "optional normal-engine trace")
	segments := flag.String("segments", "private/english-segments.json", "English segment tables")
	flag.Parse()
	rules, err := frontend.OpenEnglishDecisionTrees(*tables)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(*reference)
	if err != nil {
		return err
	}
	type decision struct {
		TreeIndex int      `json:"tree_index"`
		Strings   []string `json:"strings"`
		Integers  []int32  `json:"integers"`
		Result    string   `json:"result"`
	}
	var ref struct {
		Source string     `json:"source_sha256"`
		Cases  []decision `json:"cases"`
	}
	if err := json.Unmarshal(b, &ref); err != nil {
		return err
	}
	if ref.Source != rules.Source || len(ref.Cases) == 0 {
		return fmt.Errorf("invalid English decision reference")
	}
	check := func(c decision) error {
		if c.TreeIndex < 0 || c.TreeIndex >= 4 {
			return fmt.Errorf("invalid tree index")
		}
		got, err := rules.Trees[c.TreeIndex].Evaluate(c.Strings, c.Integers)
		if err != nil {
			return err
		}
		if got != c.Result {
			return fmt.Errorf("tree %d got %q want %q", c.TreeIndex, got, c.Result)
		}
		return nil
	}
	for i, c := range ref.Cases {
		if err := check(c); err != nil {
			return fmt.Errorf("case %d: %w", i, err)
		}
	}
	fmt.Printf("Verified %d native English decision evaluations\n", len(ref.Cases))
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
	count, models, applications := 0, 0, 0
	var results []string
	var features *frontend.EnglishDecisionFeatures
	segmentRules, err := frontend.OpenEnglishSegmentRules(*segments)
	if err != nil {
		return err
	}
	var chunk *frontend.EnglishPronunciationChunk
	step, chunks, finals := 0, 0, 0
	for _, raw := range events {
		var h struct {
			Event string `json:"event"`
		}
		if err := json.Unmarshal(raw, &h); err != nil {
			return err
		}
		switch h.Event {
		case "frontend_error", "forward_error":
			return fmt.Errorf("failed engine trace")
		case "torch_jit_result":
			models++
		case "frontend_english_initial_candidates":
			var e struct {
				Text       string                             `json:"text"`
				Candidates []frontend.EnglishInitialCandidate `json:"candidates"`
			}
			if err := json.Unmarshal(raw, &e); err != nil {
				return err
			}
			if chunk != nil && step != len(chunk.Steps) {
				return fmt.Errorf("unfinished English chunk")
			}
			c, err := segmentRules.ResolvePronunciationChunk(e.Text, rules)
			if err != nil {
				return err
			}
			chunk, step = &c, 0
			chunks++
		case "frontend_english_context":
			var e struct {
				Prepared []frontend.PreparedEnglishSegment `json:"prepared"`
				Index    int                               `json:"index"`
				OptionA  int                               `json:"option_a"`
				OptionB  int                               `json:"option_b"`
			}
			if err := json.Unmarshal(raw, &e); err != nil {
				return err
			}
			f, err := frontend.BuildEnglishDecisionFeatures(e.Prepared, e.Index, e.OptionA, e.OptionB)
			if err != nil {
				return err
			}
			if chunk == nil || step >= len(chunk.Steps) || chunk.Steps[step].Index != e.Index || !reflect.DeepEqual(chunk.Prepared, e.Prepared) || chunk.Steps[step].Features != f {
				return fmt.Errorf("English reverse loop/context differs")
			}
			features = &chunk.Steps[step].Features
		case "frontend_english_decision":
			var c decision
			if err := json.Unmarshal(raw, &c); err != nil {
				return err
			}
			if err := check(c); err != nil {
				return err
			}
			if c.TreeIndex != len(results) {
				return fmt.Errorf("unexpected decision order")
			}
			if features == nil || len(c.Strings) != 30 || len(c.Integers) != 32 {
				return fmt.Errorf("missing English feature context")
			}
			for i, want := range c.Strings {
				if features.Strings[i] != want {
					return fmt.Errorf("string feature %d got %q want %q", i, features.Strings[i], want)
				}
			}
			for i, want := range c.Integers[:31] {
				if features.Integers[i] != want {
					return fmt.Errorf("integer feature %d got %d want %d", i, features.Integers[i], want)
				}
			}
			for _, n := range rules.Trees[c.TreeIndex].Nodes {
				if n.Kind <= 0 && n.Feature == 31 {
					return fmt.Errorf("tree consumes uninitialized native slot")
				}
			}
			value, err := rules.Trees[c.TreeIndex].Evaluate(features.Strings[:], features.Integers[:])
			if err != nil {
				return err
			}
			results = append(results, value)
			count++
		case "frontend_english_rule_application":
			var e struct {
				Before frontend.EnglishInitialCandidate `json:"before"`
				After  frontend.EnglishInitialCandidate `json:"after"`
			}
			if err := json.Unmarshal(raw, &e); err != nil {
				return err
			}
			if len(results) != 4 {
				return fmt.Errorf("application missing Go decision results")
			}
			got, err := frontend.ApplyEnglishRuleResults(e.Before, [4]string(results))
			if err != nil {
				return err
			}
			if !reflect.DeepEqual(got, e.After) {
				return fmt.Errorf("Go decision/application result mismatch")
			}
			if chunk == nil || step >= len(chunk.Steps) || !reflect.DeepEqual(chunk.Steps[step].Before, e.Before) || !reflect.DeepEqual(chunk.Steps[step].Applied, e.After) || chunk.Steps[step].Results != [4]string(results) {
				return fmt.Errorf("English chunk application differs")
			}
			step++
			results = nil
			applications++
		case "frontend_english_final_candidates":
			var e struct {
				Candidates []frontend.EnglishInitialCandidate `json:"candidates"`
				Readings   []string                           `json:"readings"`
			}
			if err := json.Unmarshal(raw, &e); err != nil {
				return err
			}
			if chunk == nil || step != len(chunk.Steps) || !reflect.DeepEqual(chunk.Candidates, e.Candidates) || !reflect.DeepEqual(chunk.Readings, e.Readings) {
				return fmt.Errorf("final English chunk differs")
			}
			finals++
		}
	}
	if models == 0 || count == 0 || applications == 0 || len(results) != 0 {
		return fmt.Errorf("incomplete English decision trace")
	}
	fmt.Printf("Verified Go-built features for %d normal-engine decisions through %d candidate applications\n", count, applications)
	if chunk == nil || step != len(chunk.Steps) {
		return fmt.Errorf("incomplete English reverse loop")
	}
	if finals > 0 && finals != chunks {
		return fmt.Errorf("missing final English chunks")
	}
	fmt.Printf("Verified %d Go pronunciation loops (%d final chunk snapshots)\n", chunks, finals)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
