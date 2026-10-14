package main

import (
	"github.com/a-kaibu/aoikane/internal/frontend"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func run() error {
	table := flag.String("tables", "private/unknown-accent.json", "inspected accent rules")
	path := flag.String("reference", "private/unknown-accent-reference.json", "native unknown accent fixtures")
	kanaPath := flag.String("kana", "private/unknown-kana.json", "pronunciation conversion rules")
	trace := flag.String("trace", "", "optional normal-engine events.json")
	codecPath := flag.String("cp932", "private/cp932.json", "CP932 mapping for trace bytes")
	flag.Parse()
	rules, err := frontend.OpenUnknownAccentRules(*table)
	if err != nil {
		return err
	}
	kana, err := frontend.OpenUnknownKanaRules(*kanaPath)
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
			Source *string `json:"source"`
			Text   string  `json:"text"`
			Down   int     `json:"down"`
			Moras  int     `json:"moras"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(b, &reference); err != nil {
		return err
	}
	if reference.Source != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(reference.Cases) == 0 {
		return fmt.Errorf("invalid unknown accent reference")
	}
	for i, c := range reference.Cases {
		got, err := rules.Accent(c.Text)
		if err != nil {
			return err
		}
		if got.Down != c.Down || got.Moras != c.Moras {
			return fmt.Errorf("case %d %q: got %d/%d want %d/%d", i, c.Text, got.Down, got.Moras, c.Down, c.Moras)
		}
		if c.Source != nil {
			combined, err := frontend.CompileUnknownKana(*c.Source, kana, rules)
			if err != nil {
				return err
			}
			if combined.Pronunciation != c.Text || combined.Accent.Down != c.Down || combined.Accent.Moras != c.Moras {
				return fmt.Errorf("combined unknown kana case %d mismatch", i)
			}
		}
	}
	fmt.Printf("Verified %d native unknown-word accents and mora counts\n", len(reference.Cases))
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
			if header.Event != "frontend_unknown_accent" {
				continue
			}
			var event struct {
				Input []int `json:"input"`
				Down  int   `json:"down"`
				Moras int   `json:"moras"`
			}
			if err := json.Unmarshal(raw, &event); err != nil {
				return err
			}
			b := make([]byte, len(event.Input))
			for i, v := range event.Input {
				if v < 0 || v > 255 {
					return fmt.Errorf("invalid trace byte")
				}
				b[i] = byte(v)
			}
			text, err := codec.Decode(b)
			if err != nil {
				return err
			}
			got, err := rules.Accent(text)
			if err != nil {
				return err
			}
			if got.Down != event.Down || got.Moras != event.Moras {
				return fmt.Errorf("native accent %q mismatch", text)
			}
			count++
		}
		if count == 0 {
			return fmt.Errorf("trace contains no unknown accents")
		}
		fmt.Printf("Verified %d normal-engine unknown accents\n", count)
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
