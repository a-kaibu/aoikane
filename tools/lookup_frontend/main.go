// Inspect the real language dictionary using the portable Go reader.
package main

import (
	"github.com/a-kaibu/aoikane/internal/frontend"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func main() {
	dir := flag.String("dictionary", "private/frontend-data", "directory containing SMecab sys.dic/sys.tok/sys.fea")
	analyze := flag.Bool("analyze", false, "inspect a complete dictionary and connection-cost path")
	validate := flag.Bool("validate", false, "decode every dictionary entry and report metadata coverage")
	morphologyPath := flag.String("morphology", "", "private morphology asset for inspected word names and IDs")
	flag.Parse()
	var d *frontend.Dictionary
	var a *frontend.Analyzer
	var err error
	if *analyze {
		a, err = frontend.OpenAnalyzer(*dir)
	} else {
		d, err = frontend.OpenDictionary(*dir)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	var morphology *frontend.Morphology
	if *morphologyPath != "" {
		morphology, err = frontend.OpenMorphology(*morphologyPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if *validate {
		if d == nil {
			d = a.Dictionary
		}
		accents, connections := 0, 0
		for i := uint32(0); i < d.EntryCount(); i++ {
			e, err := d.LookupIndex(i)
			if err != nil {
				fmt.Fprintf(os.Stderr, "entry %d: %v\n", i, err)
				os.Exit(1)
			}
			for _, g := range e.AccentGroups {
				accents += len(g.Accents)
				connections += len(g.Connections)
			}
		}
		if err := encoder.Encode(map[string]any{"entries": d.EntryCount(), "accent_alternatives": accents, "accent_connections": connections}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	for _, text := range flag.Args() {
		if *analyze {
			words, err := a.Analyze(text)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			if morphology != nil {
				for i := range words {
					v, err := morphology.Resolve(words[i].Entry)
					if err != nil {
						fmt.Fprintln(os.Stderr, err)
						os.Exit(1)
					}
					words[i].Morphology = &v
				}
			}
			var unresolved []string
			for _, word := range words {
				if word.Entry.Unknown {
					unresolved = append(unresolved, word.Text)
				}
			}
			if err = encoder.Encode(struct {
				Text            string          `json:"text"`
				Pronunciation   string          `json:"pronunciation"`
				Words           []frontend.Word `json:"words"`
				UnresolvedWords []string        `json:"unresolved_words,omitempty"`
			}{text, frontend.DictionaryPronunciation(words), words, unresolved}); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			continue
		}
		entries, err := d.LookupPrefix(text)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err = encoder.Encode(struct {
			Text    string           `json:"text"`
			Entries []frontend.Entry `json:"entries"`
		}{text, entries}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
