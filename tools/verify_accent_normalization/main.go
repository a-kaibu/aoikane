package main

import (
	"github.com/a-kaibu/aoikane/internal/frontend"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	path := flag.String("reference", "private/accent-normalization-reference.json", "private native accent normalization reference")
	flag.Parse()
	b, err := os.ReadFile(*path)
	if err != nil {
		return err
	}
	type caseData struct {
		Data       string `json:"data"`
		Accent     string `json:"accent"`
		Connection string `json:"connection"`
	}
	var ref struct {
		Source   string     `json:"source_sha256"`
		Cases    []caseData `json:"cases"`
		Rejected []caseData `json:"rejected"`
	}
	if err = json.Unmarshal(b, &ref); err != nil {
		return err
	}
	if ref.Source != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(ref.Cases) == 0 {
		return fmt.Errorf("missing or incompatible references")
	}
	for i, c := range ref.Cases {
		data, err := hex.DecodeString(c.Data)
		if err != nil {
			return err
		}
		records, _, err := frontend.DecodeAccentRecord(data)
		if err != nil {
			return err
		}
		normalized, err := frontend.NormalizeAccent(records)
		if err != nil {
			return fmt.Errorf("case %d: %w", i, err)
		}
		got := strings.TrimPrefix(frontend.FormatAccentRecord(normalized), "kaccent=")
		if got != c.Accent || c.Connection != "C3" {
			return fmt.Errorf("case %d: accent %q/%q, connection %q", i, got, c.Accent, c.Connection)
		}
		// Normalizing already-explicit accent fields must be idempotent.
		again, err := frontend.NormalizeAccent(normalized)
		if err != nil {
			return err
		}
		if frontend.FormatAccentRecord(again) != frontend.FormatAccentRecord(normalized) {
			return fmt.Errorf("case %d: normalization is not idempotent", i)
		}
	}
	for i, c := range ref.Rejected {
		data, err := hex.DecodeString(c.Data)
		if err != nil {
			return err
		}
		records, _, err := frontend.DecodeAccentRecord(data)
		if err != nil {
			return err
		}
		if _, err = frontend.NormalizeAccent(records); err == nil {
			return fmt.Errorf("native rejected case %d was accepted", i)
		}
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]int{"cases": len(ref.Cases), "rejections": len(ref.Rejected)})
}
