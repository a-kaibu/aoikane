package main

import (
	"github.com/a-kaibu/aoikane/internal/frontend"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func run() error {
	table := flag.String("tables", "private/unknown-kana.json", "inspected kana rules")
	path := flag.String("reference", "private/unknown-kana-reference.json", "native conversion fixtures")
	trace := flag.String("trace", "", "optional normal-engine events.json")
	codecPath := flag.String("cp932", "private/cp932.json", "CP932 mapping for trace bytes")
	flag.Parse()
	rules, err := frontend.OpenUnknownKanaRules(*table)
	if err != nil {
		return err
	}
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
		return fmt.Errorf("invalid unknown-kana reference")
	}
	for i, c := range reference.Cases {
		got, err := rules.Pronunciation(c.Text)
		if err != nil {
			return fmt.Errorf("case %d: %w", i, err)
		}
		if got != c.Result {
			return fmt.Errorf("case %d %q: got %q want %q", i, c.Text, got, c.Result)
		}
	}
	fmt.Printf("Verified %d native unknown-kana pronunciation conversions\n", len(reference.Cases))
	if *trace != "" {
		codec, err := frontend.OpenCP932Codec(*codecPath)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(*trace)
		if err != nil {
			return err
		}
		var events []json.RawMessage
		if err := json.Unmarshal(b, &events); err != nil {
			return err
		}
		count := 0
		for _, raw := range events {
			var header struct {
				Event string `json:"event"`
			}
			if err := json.Unmarshal(raw, &header); err != nil {
				return err
			}
			if header.Event == "frontend_error" || header.Event == "forward_error" {
				return fmt.Errorf("failed engine trace")
			}
			if header.Event != "frontend_unknown_kana" {
				continue
			}
			var event struct {
				Input  []int `json:"input"`
				Result []int `json:"result"`
			}
			if err := json.Unmarshal(raw, &event); err != nil {
				return err
			}
			decode := func(values []int) (string, error) {
				b := make([]byte, len(values))
				for i, v := range values {
					if v < 0 || v > 255 {
						return "", fmt.Errorf("invalid trace byte")
					}
					b[i] = byte(v)
				}
				return codec.Decode(b)
			}
			text, err := decode(event.Input)
			if err != nil {
				return err
			}
			want, err := decode(event.Result)
			if err != nil {
				return err
			}
			got, err := rules.Pronunciation(text)
			if err != nil {
				return err
			}
			if got != want {
				return fmt.Errorf("native unknown kana %q: got %q want %q", text, got, want)
			}
			count++
		}
		if count == 0 {
			return fmt.Errorf("trace contains no unknown-kana conversions")
		}
		fmt.Printf("Verified %d normal-engine unknown-kana conversions\n", count)
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
