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
	tables := flag.String("tables", "private/english-segments.json", "inspected segmentation rules")
	reference := flag.String("reference", "private/english-segments-reference.json", "native references")
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
	var ref struct {
		Source string `json:"source_sha256"`
		Cases  []struct {
			Text     string                            `json:"text"`
			Units    []frontend.EnglishSegment         `json:"units"`
			Prepared []frontend.PreparedEnglishSegment `json:"prepared"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(b, &ref); err != nil {
		return err
	}
	if ref.Source != rules.Source || len(ref.Cases) == 0 {
		return fmt.Errorf("invalid segmentation reference")
	}
	for i, c := range ref.Cases {
		got, err := rules.Segment(c.Text)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(got, c.Units) {
			return fmt.Errorf("case %d %q got %+v want %+v", i, c.Text, got, c.Units)
		}
		prepared, err := rules.Prepare(got)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(prepared, c.Prepared) {
			return fmt.Errorf("preparation %d %q got %+v want %+v", i, c.Text, prepared, c.Prepared)
		}
	}
	fmt.Printf("Verified %d native English segmentations and preparations\n", len(ref.Cases))
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
	count, models, preparations, initialPasses, initialUnits, firstContexts := 0, 0, 0, 0, 0, 0
	var pendingInitial []frontend.PreparedEnglishSegment
	applications, changed := 0, 0
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
		case "frontend_english_rule_application":
			var e struct {
				Before  frontend.EnglishInitialCandidate `json:"before"`
				After   frontend.EnglishInitialCandidate `json:"after"`
				Results []string                         `json:"results"`
			}
			if err := json.Unmarshal(raw, &e); err != nil {
				return err
			}
			if len(e.Results) != 4 {
				return fmt.Errorf("invalid English decision result count")
			}
			got, err := frontend.ApplyEnglishRuleResults(e.Before, [4]string(e.Results))
			if err != nil {
				return err
			}
			if !reflect.DeepEqual(got, e.After) {
				return fmt.Errorf("English application got %+v want %+v", got, e.After)
			}
			if !reflect.DeepEqual(e.Before, e.After) {
				changed++
			}
			applications++
		case "frontend_english_initial_candidates":
			var e struct {
				Text           string                             `json:"text"`
				Units          []frontend.EnglishSegment          `json:"units"`
				Candidates     []frontend.EnglishInitialCandidate `json:"candidates"`
				ConditionUnits []frontend.PreparedEnglishSegment  `json:"condition_units"`
			}
			if err := json.Unmarshal(raw, &e); err != nil {
				return err
			}
			units, err := rules.Segment(e.Text)
			if err != nil {
				return err
			}
			if !reflect.DeepEqual(units, e.Units) {
				return fmt.Errorf("candidate segmentation %q mismatch", e.Text)
			}
			got, err := rules.InitialCandidates(units)
			if err != nil {
				return err
			}
			if !reflect.DeepEqual(got, e.Candidates) {
				return fmt.Errorf("initial candidates %q got %+v want %+v", e.Text, got, e.Candidates)
			}
			condition, err := rules.Prepare(units)
			if err != nil {
				return err
			}
			if !reflect.DeepEqual(condition, e.ConditionUnits) {
				return fmt.Errorf("initial condition units %q mismatch", e.Text)
			}
			pendingInitial = condition
			initialPasses++
			initialUnits += len(units)
		case "frontend_english_context":
			if pendingInitial != nil {
				var e struct {
					Prepared []frontend.PreparedEnglishSegment `json:"prepared"`
					Index    int                               `json:"index"`
					OptionA  int                               `json:"option_a"`
					OptionB  int                               `json:"option_b"`
					Context  frontend.EnglishConditionContext  `json:"context"`
				}
				if err := json.Unmarshal(raw, &e); err != nil {
					return err
				}
				if !reflect.DeepEqual(pendingInitial, e.Prepared) {
					return fmt.Errorf("first context input differs from initial candidates")
				}
				got, err := frontend.BuildEnglishConditionContext(pendingInitial, e.Index, e.OptionA, e.OptionB)
				if err != nil {
					return err
				}
				if got != e.Context {
					return fmt.Errorf("first context after candidate setup mismatch")
				}
				pendingInitial = nil
				firstContexts++
			}
		case "frontend_english_preparation":
			var e struct {
				Unit     frontend.EnglishSegment         `json:"unit"`
				Prepared frontend.PreparedEnglishSegment `json:"prepared"`
			}
			if err := json.Unmarshal(raw, &e); err != nil {
				return err
			}
			got, err := rules.Prepare([]frontend.EnglishSegment{e.Unit})
			if err != nil {
				return err
			}
			if len(got) != 1 || got[0] != e.Prepared {
				return fmt.Errorf("normal preparation mismatch")
			}
			preparations++
		case "frontend_english_segments":
			var e struct {
				Text  string                    `json:"text"`
				Units []frontend.EnglishSegment `json:"units"`
			}
			if err := json.Unmarshal(raw, &e); err != nil {
				return err
			}
			got, err := rules.Segment(e.Text)
			if err != nil {
				return err
			}
			if !reflect.DeepEqual(got, e.Units) {
				return fmt.Errorf("normal segmentation %q mismatch", e.Text)
			}
			count++
		}
	}
	if count == 0 || models == 0 || preparations == 0 {
		return fmt.Errorf("incomplete normal segmentation trace")
	}
	fmt.Printf("Verified %d normal-engine English segmentations and %d unit preparations\n", count, preparations)
	if initialPasses > 0 {
		if firstContexts != initialPasses {
			return fmt.Errorf("initial candidate passes did not all reach a context")
		}
		fmt.Printf("Verified %d initial candidate passes (%d units) through their first pronunciation contexts\n", initialPasses, initialUnits)
	}
	if applications > 0 {
		fmt.Printf("Verified %d English rule applications (%d changed candidates)\n", applications, changed)
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
