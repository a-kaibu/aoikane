package frontend

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

// PhoneDefinition describes a phoneme before contextual allophone and prosody
// rules run. Art is the base bitmap; it is not a complete FeatNet input.
type PhoneDefinition struct {
	Label string `json:"label"`
	ID    int    `json:"id"`
	Info  uint16 `json:"info"`
	Art   uint32 `json:"art"`
}

type moraEntry struct {
	Mora   string `json:"mora"`
	Phones []int  `json:"phones"`
}

type moraGroup struct {
	ExpandedOnly bool        `json:"expanded_only"`
	Entries      []moraEntry `json:"entries"`
}

// MoraEncoder implements the inspected table lookup, not full Japanese text
// analysis. Expanded selects the native lookup's alternate glide decomposition.
// Its use by the surrounding original language pipeline remains to be verified.
type MoraEncoder struct {
	Version      int               `json:"version"`
	SourceSHA256 string            `json:"source_sha256"`
	Phones       []PhoneDefinition `json:"phones"`
	Groups       []moraGroup       `json:"groups"`
}

func OpenMoraEncoder(path string) (*MoraEncoder, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var e MoraEncoder
	if err := json.Unmarshal(b, &e); err != nil {
		return nil, fmt.Errorf("phonetics asset: %w", err)
	}
	if e.Version != 1 || len(e.Phones) != 55 || len(e.Groups) != 10 {
		return nil, fmt.Errorf("unsupported phonetics asset")
	}
	for _, p := range e.Phones {
		if p.ID < 0 || p.ID >= 52 || p.Label == "" {
			return nil, fmt.Errorf("invalid phoneme definition")
		}
	}
	for _, g := range e.Groups {
		if len(g.Entries) == 0 {
			return nil, fmt.Errorf("empty mora group")
		}
		for _, entry := range g.Entries {
			if !utf8.ValidString(entry.Mora) || entry.Mora == "" ||
				len(entry.Phones) == 0 || len(entry.Phones) > 4 {
				return nil, fmt.Errorf("invalid mora entry")
			}
			for _, p := range entry.Phones {
				if p < 0 || p >= len(e.Phones) {
					return nil, fmt.Errorf("mora entry references invalid phoneme")
				}
			}
		}
	}
	return &e, nil
}

// LookupPrefix preserves the native group order: some identical keys have
// different decompositions in different groups. Unknown input returns nil, 0.
// Returned indices select Phones; nasalized g entries can share a minimal ID.
func (e *MoraEncoder) LookupPrefix(text string, expanded bool) ([]int, int) {
	for _, g := range e.Groups {
		if g.ExpandedOnly && !expanded {
			continue
		}
		for _, entry := range g.Entries {
			if strings.HasPrefix(text, entry.Mora) {
				return append([]int(nil), entry.Phones...), len(entry.Mora)
			}
		}
	}
	return nil, 0
}
