package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/a-kaibu/aoikane/internal/frontend"
)

func run() error {
	path := flag.String("reference", "private/text-normalization-reference.json", "native normalization fixtures")
	trace := flag.String("trace", "", "optional normal-engine events.json")
	flag.Parse()
	b, err := os.ReadFile(*path)
	if err != nil {
		return err
	}
	var reference struct {
		Source string `json:"source_sha256"`
		Cases  []struct {
			Text   string `json:"text"`
			Result string `json:"result"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(b, &reference); err != nil {
		return err
	}
	if reference.Source != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(reference.Cases) == 0 {
		return fmt.Errorf("invalid normalization reference")
	}
	for i, c := range reference.Cases {
		got, err := frontend.NormalizeText(c.Text)
		if err != nil {
			return fmt.Errorf("case %d: %w", i, err)
		}
		if got != c.Result {
			return fmt.Errorf("case %d input %q: got %q want %q", i, c.Text, got, c.Result)
		}
	}
	traceCount := 0
	if *trace != "" {
		b, err := os.ReadFile(*trace)
		if err != nil {
			return err
		}
		var events []json.RawMessage
		if err := json.Unmarshal(b, &events); err != nil {
			return err
		}
		for _, raw := range events {
			var header struct {
				Event string `json:"event"`
			}
			if err := json.Unmarshal(raw, &header); err != nil {
				return err
			}
			if header.Event == "frontend_error" || header.Event == "forward_error" {
				return fmt.Errorf("failed normalization trace")
			}
			if header.Event != "frontend_normalization" {
				continue
			}
			var event struct {
				Input    string `json:"input"`
				Result   string `json:"result"`
				Consumed int    `json:"consumed"`
			}
			if err := json.Unmarshal(raw, &event); err != nil {
				return err
			}
			got, err := frontend.NormalizeText(event.Input)
			if err != nil {
				return err
			}
			if got != event.Result || len(event.Input) != event.Consumed {
				return fmt.Errorf("trace normalization mismatch: %q got %q want %q", event.Input, got, event.Result)
			}
			traceCount++
		}
		if traceCount == 0 {
			return fmt.Errorf("trace contains no normalization observations")
		}
	}
	fmt.Printf("Verified %d native text normalization cases\n", len(reference.Cases))
	if traceCount > 0 {
		fmt.Printf("Verified %d normal-engine normalization observations\n", traceCount)
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
