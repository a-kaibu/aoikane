package frontend

import (
	"encoding/json"
	"fmt"
	"os"
	"unicode/utf8"
)

type MorphologyValue struct {
	ID   uint32 `json:"id"`
	Name string `json:"name"`
}

type Morphology struct {
	Version      int               `json:"version"`
	SourceSHA256 string            `json:"source_sha256"`
	POS          []MorphologyValue `json:"pos"`
	Types        []MorphologyValue `json:"types"`
	Forms        []MorphologyValue `json:"forms"`
}

type WordMorphology struct {
	POS  MorphologyValue `json:"pos"`
	Type MorphologyValue `json:"type"`
	Form MorphologyValue `json:"form"`
}

func OpenMorphology(path string) (*Morphology, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Morphology
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	if m.Version != 1 || m.SourceSHA256 != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(m.POS) != 79 || len(m.Types) != 77 || len(m.Forms) != 33 {
		return nil, fmt.Errorf("unsupported morphology asset")
	}
	for _, table := range [][]MorphologyValue{m.POS, m.Types, m.Forms} {
		if table[0].ID != 0 {
			return nil, fmt.Errorf("morphology table lacks its default entry")
		}
		for _, v := range table {
			if !utf8.ValidString(v.Name) {
				return nil, fmt.Errorf("invalid morphology name UTF-8")
			}
		}
	}
	return &m, nil
}

// Lookup reproduces the native small-code mapping and default for out-of-range
// bytes. The structured IDs are used by language predicates; names are for
// inspection. Neither a text model nor the original engine is involved.
func (m *Morphology) Lookup(kind string, code byte) (MorphologyValue, error) {
	var table []MorphologyValue
	switch kind {
	case "pos":
		table = m.POS
	case "types":
		table = m.Types
	case "forms":
		table = m.Forms
	default:
		return MorphologyValue{}, fmt.Errorf("unknown morphology table %q", kind)
	}
	if len(table) == 0 {
		return MorphologyValue{}, fmt.Errorf("empty morphology table")
	}
	if int(code) >= len(table) {
		return table[0], nil
	}
	return table[code], nil
}

func (m *Morphology) Resolve(e Entry) (WordMorphology, error) {
	p, err := m.Lookup("pos", e.POS)
	if err != nil {
		return WordMorphology{}, err
	}
	t, err := m.Lookup("types", e.ConjugationType)
	if err != nil {
		return WordMorphology{}, err
	}
	f, err := m.Lookup("forms", e.Form)
	if err != nil {
		return WordMorphology{}, err
	}
	return WordMorphology{POS: p, Type: t, Form: f}, nil
}
