// Inspect reconstructed dictionary analysis and reading candidate selection.
// This is not yet a complete phrase/accent frontend.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/a-kaibu/aoikane/internal/frontend"
)

func run() error {
	directory := flag.String("dictionary", "private/frontend-data", "language dictionary directory")
	morphologyPath := flag.String("morphology", "private/morphology.json", "structured morphology IDs")
	rulesPath := flag.String("rules", "private/reading-rules-utf8.txt", "UTF-8 reading rules")
	codecPath := flag.String("cp932", "private/cp932.json", "portable Windows CP932 mappings")
	flag.Parse()
	if flag.NArg() == 0 {
		return fmt.Errorf("provide input text")
	}
	analyzer, err := frontend.OpenAnalyzer(*directory)
	if err != nil {
		return err
	}
	morphology, err := frontend.OpenMorphology(*morphologyPath)
	if err != nil {
		return err
	}
	f, err := os.Open(*rulesPath)
	if err != nil {
		return err
	}
	defer f.Close()
	rules, err := frontend.ParseReadingRules(f)
	if err != nil {
		return err
	}
	codec, err := frontend.OpenCP932Codec(*codecPath)
	if err != nil {
		return err
	}
	for _, text := range flag.Args() {
		normalized, err := frontend.NormalizeText(text)
		if err != nil {
			return err
		}
		words, err := analyzer.Analyze(normalized)
		if err != nil {
			return err
		}
		context := make([]frontend.LanguageWord, len(words))
		surfaces := make([]string, len(words))
		for i, w := range words {
			surfaces[i] = w.Text
			if w.Entry.Unknown {
				return fmt.Errorf("reading for unknown word %q is not reconstructed", w.Text)
			}
			context[i], err = morphology.LanguageWord(w)
			if err != nil {
				return err
			}
		}
		contextText, err := codec.ReadingContextText(text, surfaces)
		if err != nil {
			return err
		}
		var result []struct {
			Text          string                    `json:"text"`
			Selection     frontend.ReadingSelection `json:"selection"`
			Pronunciation string                    `json:"pronunciation"`
			Accent        string                    `json:"accent"`
			Connection    string                    `json:"connection"`
			ReadingAccent frontend.ReadingAccent    `json:"reading_accent"`
		}
		for i, w := range words {
			candidate, selection, err := rules.SelectDictionaryCandidate(context, i, w.Entry, contextText)
			if err != nil {
				return fmt.Errorf("word %q: %w", w.Text, err)
			}
			readingAccent, err := frontend.UpdateReadingAccent(frontend.ReadingAccent{}, &candidate, codec)
			if err != nil {
				return fmt.Errorf("word %q accent: %w", w.Text, err)
			}
			result = append(result, struct {
				Text          string                    `json:"text"`
				Selection     frontend.ReadingSelection `json:"selection"`
				Pronunciation string                    `json:"pronunciation"`
				Accent        string                    `json:"accent"`
				Connection    string                    `json:"connection"`
				ReadingAccent frontend.ReadingAccent    `json:"reading_accent"`
			}{w.Text, selection, candidate.Pronunciation, candidate.Accent, candidate.Connection, readingAccent})
		}
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
