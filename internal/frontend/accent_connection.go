package frontend

import (
	"fmt"
	"strings"
)

// AccentConnection preserves a dictionary rule, its preceding-word condition,
// signed parameter and optional alternative selected by an accent comparison.
// Applying P/C/F/S/K/A/CF rules to a phrase is separate from decoding them.
type AccentConnection struct {
	Rule       int               `json:"rule"`
	Condition  int               `json:"condition"`
	Offset     *int              `json:"offset,omitempty"`
	Comparison int               `json:"comparison,omitempty"`
	Threshold  int               `json:"threshold,omitempty"`
	Otherwise  *AccentConnection `json:"otherwise,omitempty"`
}

func signedNibble(b byte) int {
	v := int(b & 15)
	if v >= 8 {
		v -= 16
	}
	return v
}

func DecodeAccentConnections(data []byte) ([]AccentConnection, int, error) {
	var records []AccentConnection
	at := 0
	for len(records) < 256 {
		if at >= len(data) {
			return nil, 0, fmt.Errorf("truncated accent connection record")
		}
		code := data[at]
		need := 1
		if code&2 != 0 {
			need++
		}
		if code&1 != 0 {
			need += 2
		}
		if len(data)-at < need {
			return nil, 0, fmt.Errorf("truncated accent connection parameters")
		}
		b := data[at : at+need]
		r := AccentConnection{Rule: int(code >> 2)}
		next := 1
		if code&2 != 0 {
			r.Condition = int(b[1] >> 5)
			if b[1]&16 != 0 {
				v := signedNibble(b[1])
				r.Offset = &v
			}
			next++
		}
		if code&1 != 0 {
			r.Comparison = int(b[next+1] >> 6)
			r.Threshold = int(b[next+1] >> 3 & 7)
			r.Otherwise = &AccentConnection{Rule: int(b[next]>>5) | int(b[next+1]&7)<<3}
			if b[next]&16 != 0 {
				v := signedNibble(b[next])
				r.Otherwise.Offset = &v
			}
		}
		records = append(records, r)
		at += need
		if r.Condition == 0 {
			return records, at, nil
		}
	}
	return nil, 0, fmt.Errorf("too many accent connection alternatives")
}

var connectionNames = [...]string{
	"P1", "P2", "P3", "P4", "C1", "C2", "C3", "C4",
	"F1", "F2", "F3", "F4", "F5", "F6", "S1", "S2", "S3",
	"K1", "K2", "K3", "K4", "K5", "K6", "K7", "K8", "A1",
	"CF1", "CF2", "CF3", "CF4", "CF5", "CF6", "*",
}

// FormatAccentConnections reproduces the native rule inspector. KanaAccent
// selects its kaccent_con header. Unknown rule IDs have no printed name, matching
// the original inspector; callers must validate them before executing rules.
func FormatAccentConnections(records []AccentConnection, kanaAccent bool) string {
	if len(records) == 0 {
		return ""
	}
	conditions := [...]string{"", "名詞", "動詞", "形容詞", "特殊助動詞", "連体修飾節"}
	comparisons := [...]string{"eq", "le", "ge", ""}
	var b strings.Builder
	if kanaAccent {
		b.WriteByte('k')
	}
	b.WriteString("accent_con=")
	writeRule := func(r AccentConnection) {
		if r.Rule >= 0 && r.Rule < len(connectionNames) {
			b.WriteString(connectionNames[r.Rule])
		}
		if r.Offset != nil {
			fmt.Fprintf(&b, "@%d", *r.Offset)
		}
	}
	for i, r := range records {
		if i != 0 {
			b.WriteByte(',')
		}
		if r.Condition > 0 && r.Condition < len(conditions) {
			b.WriteString(conditions[r.Condition])
			b.WriteByte('%')
		}
		writeRule(r)
		if r.Otherwise != nil {
			if r.Comparison >= 0 && r.Comparison < len(comparisons) {
				b.WriteString(comparisons[r.Comparison])
			}
			fmt.Fprintf(&b, "%delse", r.Threshold)
			writeRule(*r.Otherwise)
		}
	}
	return strings.TrimSuffix(b.String(), ",*")
}
