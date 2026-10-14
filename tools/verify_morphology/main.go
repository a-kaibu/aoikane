// Compare portable POS/conjugation lookup against all native byte inputs.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/a-kaibu/aoikane/internal/frontend"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	asset := flag.String("asset", "private/morphology.json", "private morphology tables")
	path := flag.String("reference", "private/morphology-reference.json", "native lookup references")
	flag.Parse()
	m, err := frontend.OpenMorphology(*asset)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(*path)
	if err != nil {
		return err
	}
	var ref struct {
		Source string `json:"source_sha256"`
		Cases  []struct {
			Kind string `json:"kind"`
			Code int    `json:"code"`
			ID   uint32 `json:"id"`
			Name string `json:"name"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(b, &ref); err != nil {
		return err
	}
	if ref.Source != m.SourceSHA256 || len(ref.Cases) != 768 {
		return fmt.Errorf("missing or incompatible morphology references")
	}
	seen := map[string]map[int]bool{}
	for i, c := range ref.Cases {
		if c.Code < 0 || c.Code > 255 {
			return fmt.Errorf("invalid byte code in case %d", i)
		}
		got, err := m.Lookup(c.Kind, byte(c.Code))
		if err != nil {
			return err
		}
		if got.ID != c.ID || got.Name != c.Name {
			return fmt.Errorf("case %d %s/%d: got %#v want %08x %q", i, c.Kind, c.Code, got, c.ID, c.Name)
		}
		if seen[c.Kind] == nil {
			seen[c.Kind] = map[int]bool{}
		}
		if seen[c.Kind][c.Code] {
			return fmt.Errorf("duplicate reference %s/%d", c.Kind, c.Code)
		}
		seen[c.Kind][c.Code] = true
	}
	for _, kind := range []string{"pos", "types", "forms"} {
		if len(seen[kind]) != 256 {
			return fmt.Errorf("missing %s byte cases", kind)
		}
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]int{"cases": len(ref.Cases)})
}
