package frontend

import (
	"fmt"
	"strings"
)

// AccentGroup corresponds to one pronunciation alternative. Each alternative
// has its own accent candidates and connection conditions; selecting a reading
// must select this associated group too.
type AccentGroup struct {
	Accents     []Accent           `json:"accents"`
	Connections []AccentConnection `json:"connections"`
}

func DecodeAccentGroups(data []byte) ([]AccentGroup, int, error) {
	var groups []AccentGroup
	at := 0
	for len(groups) < 256 {
		if at >= len(data) {
			return nil, 0, fmt.Errorf("truncated pronunciation accent group")
		}
		more := data[at]&2 != 0
		accents, n, err := DecodeAccentRecord(data[at:])
		if err != nil {
			return nil, 0, err
		}
		at += n
		connections, n, err := DecodeAccentConnections(data[at:])
		if err != nil {
			return nil, 0, err
		}
		at += n
		groups = append(groups, AccentGroup{Accents: accents, Connections: connections})
		if !more {
			return groups, at, nil
		}
	}
	return nil, 0, fmt.Errorf("too many pronunciation accent groups")
}

func FormatAccentGroups(groups []AccentGroup) string {
	var b strings.Builder
	if len(groups) > 1 {
		b.WriteByte('{')
	}
	for i, g := range groups {
		if i != 0 {
			b.WriteByte('/')
		}
		b.WriteString(FormatAccentRecord(g.Accents))
		b.WriteByte(':')
		kana := len(g.Accents) > 0 && g.Accents[0].Code&64 != 0
		b.WriteString(FormatAccentConnections(g.Connections, kana))
	}
	if len(groups) > 1 {
		b.WriteByte('}')
	}
	return b.String()
}
