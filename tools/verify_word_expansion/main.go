package main

import (
	"github.com/a-kaibu/aoikane/internal/frontend"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"reflect"
)

type word struct {
	Previous, Next, First, Last int
	Surface, Annotation         string
	Category                    int32
}

type expansionCase struct {
	Before, After []word
	Returned      int
}

func run() error {
	path := flag.String("reference", "private/word-expansion-reference.json", "native expansion reference")
	trace := flag.String("trace", "", "optional normal synthesis trace")
	flag.Parse()
	data, err := os.ReadFile(*path)
	if err != nil {
		return err
	}
	var ref struct {
		Source string `json:"source_sha256"`
		Cases  []expansionCase
	}
	if err := json.Unmarshal(data, &ref); err != nil {
		return err
	}
	if ref.Source != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(ref.Cases) == 0 {
		return fmt.Errorf("invalid expansion reference")
	}
	referenceCount := len(ref.Cases)
	expanded, models := 0, 0
	if *trace != "" {
		data, err = os.ReadFile(*trace)
		if err != nil {
			return err
		}
		var events []json.RawMessage
		if err := json.Unmarshal(data, &events); err != nil {
			return err
		}
		for _, raw := range events {
			var e struct{ Event string }
			if err := json.Unmarshal(raw, &e); err != nil {
				return err
			}
			switch e.Event {
			case "frontend_error", "forward_error":
				return fmt.Errorf("failed native trace")
			case "torch_jit_result":
				models++
			case "frontend_word_expansion":
				var c expansionCase
				if err := json.Unmarshal(raw, &c); err != nil {
					return err
				}
				if len(c.Before) > 0 && c.Before[0].First >= 0 {
					expanded++
				}
				ref.Cases = append(ref.Cases, c)
			}
		}
		if models == 0 || len(ref.Cases) == referenceCount {
			return fmt.Errorf("incomplete expansion trace")
		}
	}
	for i, c := range ref.Cases {
		if len(c.Before) < 2 || len(c.After) != len(c.Before) {
			return fmt.Errorf("invalid case %d", i)
		}
		nodes := make([]frontend.ExpansionWord, len(c.Before))
		indices := make(map[*frontend.ExpansionWord]int)
		for j := range nodes {
			indices[&nodes[j]] = j
		}
		link := func(index int) (*frontend.ExpansionWord, error) {
			if index == -1 {
				return nil, nil
			}
			if index < 0 || index >= len(nodes) {
				return nil, fmt.Errorf("invalid node index %d", index)
			}
			return &nodes[index], nil
		}
		for j, w := range c.Before {
			n := &nodes[j]
			n.Surface, n.Annotation, n.Category = w.Surface, w.Annotation, w.Category
			for _, l := range []struct {
				target **frontend.ExpansionWord
				index  int
			}{{&n.Previous, w.Previous}, {&n.Next, w.Next}, {&n.ExpansionFirst, w.First}, {&n.ExpansionLast, w.Last}} {
				*l.target, err = link(l.index)
				if err != nil {
					return err
				}
			}
		}
		last, err := frontend.ExpandWordChain(&nodes[0])
		if err != nil {
			return fmt.Errorf("case %d: %w", i, err)
		}
		if last == nil || indices[last] != c.Returned {
			return fmt.Errorf("case %d returned wrong node", i)
		}
		index := func(n *frontend.ExpansionWord) int {
			if n == nil {
				return -1
			}
			return indices[n]
		}
		for j, n := range nodes {
			got := word{Previous: index(n.Previous), Next: index(n.Next), First: index(n.ExpansionFirst), Last: index(n.ExpansionLast), Surface: n.Surface, Annotation: n.Annotation, Category: n.Category}
			if !reflect.DeepEqual(got, c.After[j]) {
				return fmt.Errorf("case %d node %d got %+v want %+v", i, j, got, c.After[j])
			}
		}
	}
	fmt.Printf("Verified %d reference expansions and %d synthesis calls (%d with detached chains), including links, boundary and return node\n", referenceCount, len(ref.Cases)-referenceCount, expanded)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
