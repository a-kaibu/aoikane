// Compare the Go contextual devoicing pass against independent native cases.
package main

import (
	"github.com/a-kaibu/aoikane/internal/frontend"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"reflect"
)

func run() error {
	path := flag.String("reference", "private/devoicing-reference.json", "native reference JSON")
	flag.Parse()
	b, err := os.ReadFile(*path)
	if err != nil {
		return err
	}
	var ref struct {
		Version int    `json:"version"`
		SHA     string `json:"source_sha256"`
		Cases   []struct {
			Flags   uint32                  `json:"flags"`
			Inputs  []frontend.DevoicePhone `json:"inputs"`
			Outputs []frontend.DevoicePhone `json:"outputs"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(b, &ref); err != nil {
		return err
	}
	if ref.Version != 1 || ref.SHA != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(ref.Cases) == 0 {
		return fmt.Errorf("invalid native reference")
	}
	phones, changed := 0, 0
	for ci, c := range ref.Cases {
		got := append([]frontend.DevoicePhone(nil), c.Inputs...)
		if err := frontend.ApplyDevoicing(got, c.Flags); err != nil {
			return fmt.Errorf("case %d: %w", ci, err)
		}
		if !reflect.DeepEqual(got, c.Outputs) {
			return fmt.Errorf("case %d flags %#x mismatch: input=%+v got=%+v want=%+v", ci, c.Flags, c.Inputs, got, c.Outputs)
		}
		for pi, p := range got {
			phones++
			if p != c.Inputs[pi] {
				changed++
			}
		}
	}
	fmt.Printf("Verified %d native devoicing cases, %d phones and %d changed records: labels and Info match exactly\n", len(ref.Cases), phones, changed)
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
