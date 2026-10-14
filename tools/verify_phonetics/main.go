// Compare portable mora lookup against calls to the inspected native routine.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"slices"

	"github.com/a-kaibu/aoikane/internal/frontend"
)

func main() {
	asset := flag.String("asset", "private/phonetics.json", "exported private phonetics asset")
	reference := flag.String("reference", "private/phonetics-reference.json", "native lookup cases")
	flag.Parse()
	if err := run(*asset, *reference); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(asset, reference string) error {
	e, err := frontend.OpenMoraEncoder(asset)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(reference)
	if err != nil {
		return err
	}
	var expected struct {
		SourceSHA256 string `json:"source_sha256"`
		Cases        []struct {
			Text     string `json:"text"`
			Expanded bool   `json:"expanded"`
			Bytes    int    `json:"bytes"`
			Phones   []int  `json:"phones"`
		} `json:"cases"`
		Allophones []struct {
			Input  []frontend.Allophone `json:"input"`
			Output []frontend.Allophone `json:"output"`
		} `json:"allophones"`
	}
	if err := json.Unmarshal(b, &expected); err != nil {
		return err
	}
	if expected.SourceSHA256 != e.SourceSHA256 || len(expected.Cases) == 0 {
		return fmt.Errorf("empty reference or mismatched engine version")
	}
	for _, c := range expected.Cases {
		phones, bytes := e.LookupPrefix(c.Text, c.Expanded)
		if bytes != c.Bytes || !slices.Equal(phones, c.Phones) {
			return fmt.Errorf("%q expanded=%t: got %v/%d, native %v/%d",
				c.Text, c.Expanded, phones, bytes, c.Phones, c.Bytes)
		}
	}
	if len(expected.Allophones) == 0 {
		return fmt.Errorf("no native consonant reference cases")
	}
	for i, c := range expected.Allophones {
		actual := slices.Clone(c.Input)
		if err := frontend.ApplyAllophones(actual); err != nil {
			return err
		}
		if !slices.Equal(actual, c.Output) {
			return fmt.Errorf("allophone case %d: input %v, got %v, native %v", i, c.Input, actual, c.Output)
		}
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{
		"cases": len(expected.Cases), "allophone_cases": len(expected.Allophones), "close": true,
	})
}
