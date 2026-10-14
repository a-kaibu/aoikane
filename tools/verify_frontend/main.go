// Compare the portable dictionary reader against recorded native lookups.
package main

import (
	"github.com/a-kaibu/aoikane/internal/frontend"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

type match struct {
	Index  uint32 `json:"index"`
	Length int    `json:"length"`
	Count  int    `json:"count"`
}
type event struct {
	Event   string  `json:"event"`
	Text    string  `json:"text"`
	Matches []match `json:"matches"`
	Index   uint32  `json:"index"`
	Length  int     `json:"length"`
	Count   int     `json:"count"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	dir := flag.String("dictionary", "private/frontend-data", "directory containing SMecab sys.dic/sys.tok/sys.fea")
	trace := flag.String("trace", "private/lexical-trace", "recorded native dictionary lookups")
	flag.Parse()
	d, err := frontend.OpenDictionary(*dir)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(filepath.Join(*trace, "events.json"))
	if err != nil {
		return err
	}
	var events []event
	if err = json.Unmarshal(b, &events); err != nil {
		return err
	}
	groups := map[[2]uint32]int{}
	for _, e := range events {
		if e.Event != "dictionary_group" {
			continue
		}
		key := [2]uint32{e.Index, uint32(e.Length)}
		if n, ok := groups[key]; ok && n != e.Count {
			return fmt.Errorf("conflicting native dictionary groups")
		}
		groups[key] = e.Count
	}
	calls, entries := 0, 0
	for _, e := range events {
		if e.Event != "dictionary_prefix" {
			continue
		}
		got, err := d.LookupPrefix(e.Text)
		if err != nil {
			return fmt.Errorf("lookup %q: %w", e.Text, err)
		}
		at := 0
		for _, m := range e.Matches {
			n, ok := groups[[2]uint32{m.Index, uint32(m.Length)}]
			if !ok || n <= 0 {
				return fmt.Errorf("missing native group for %q", e.Text)
			}
			for i := 0; i < n; i++ {
				if at >= len(got) || got[at].Index != m.Index+uint32(i) || got[at].Length != m.Length {
					return fmt.Errorf("dictionary mismatch for %q at alternative %d", e.Text, at)
				}
				at++
			}
		}
		if at != len(got) {
			return fmt.Errorf("extra Go dictionary matches for %q", e.Text)
		}
		calls++
		entries += len(got)
	}
	if calls == 0 || entries == 0 {
		return fmt.Errorf("no native dictionary lookups to verify")
	}
	result := struct {
		Calls        int  `json:"calls"`
		Alternatives int  `json:"alternatives"`
		Close        bool `json:"close"`
	}{calls, entries, true}
	b, err = json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(*trace, "go-dictionary-results.json"), b, 0600); err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}
