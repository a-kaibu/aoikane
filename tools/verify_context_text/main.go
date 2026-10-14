package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/a-kaibu/aoikane/internal/frontend"
)

type contextCase struct {
	Words       [][]int `json:"words"`
	SourceBytes int     `json:"source_bytes"`
	Result      []int   `json:"result"`
}

func octets(values []int) ([]byte, error) {
	b := make([]byte, len(values))
	for i, value := range values {
		if value < 0 || value > 255 {
			return nil, fmt.Errorf("invalid byte %d", value)
		}
		b[i] = byte(value)
	}
	return b, nil
}

func run() error {
	path := flag.String("reference", "private/context-text-reference.json", "native context fixtures")
	trace := flag.String("trace", "", "optional normal-engine events.json")
	flag.Parse()
	b, err := os.ReadFile(*path)
	if err != nil {
		return err
	}
	var reference struct {
		Source string        `json:"source_sha256"`
		Cases  []contextCase `json:"cases"`
	}
	if err := json.Unmarshal(b, &reference); err != nil {
		return err
	}
	if reference.Source != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(reference.Cases) == 0 {
		return fmt.Errorf("invalid document context reference")
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
			var event struct {
				Event string `json:"event"`
			}
			if err := json.Unmarshal(raw, &event); err != nil {
				return err
			}
			if event.Event == "frontend_error" || event.Event == "forward_error" {
				return fmt.Errorf("failed engine trace")
			}
			if event.Event == "frontend_context_build" {
				var c contextCase
				if err := json.Unmarshal(raw, &c); err != nil {
					return err
				}
				reference.Cases = append(reference.Cases, c)
				traceCount++
			}
		}
		if traceCount == 0 {
			return fmt.Errorf("trace contains no context build events")
		}
	}
	for i, c := range reference.Cases {
		words := make([][]byte, len(c.Words))
		for j, w := range c.Words {
			words[j], err = octets(w)
			if err != nil {
				return err
			}
		}
		want, err := octets(c.Result)
		if err != nil {
			return err
		}
		got, err := frontend.BuildReadingContext(words, c.SourceBytes)
		if err != nil {
			return fmt.Errorf("case %d: %w", i, err)
		}
		if !bytes.Equal(got, want) {
			return fmt.Errorf("case %d: got %x want %x", i, got, want)
		}
	}
	fmt.Printf("Verified %d context buffers (%d normal-engine observations)\n", len(reference.Cases), traceCount)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
