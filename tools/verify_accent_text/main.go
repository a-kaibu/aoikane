package main

import (
	"github.com/a-kaibu/aoikane/internal/frontend"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	path := flag.String("reference", "private/accent-text-reference.json", "native records")
	flag.Parse()
	b, err := os.ReadFile(*path)
	if err != nil {
		return err
	}
	var ref struct {
		Source string `json:"source_sha256"`
		Cases  []struct {
			Text   string                    `json:"text"`
			Start  int                       `json:"start"`
			Result frontend.AccentTextRecord `json:"result"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(b, &ref); err != nil {
		return err
	}
	if ref.Source != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(ref.Cases) == 0 {
		return fmt.Errorf("invalid reference")
	}
	for i, c := range ref.Cases {
		got, err := frontend.ParseAccentText(c.Text, c.Start)
		if err != nil {
			return fmt.Errorf("case %d: %w", i, err)
		}
		if got != c.Result {
			return fmt.Errorf("case %d %q at %d got %+v want %+v", i, c.Text, c.Start, got, c.Result)
		}
	}
	fmt.Printf("Verified %d native accent text records\n", len(ref.Cases))
	return nil
}
