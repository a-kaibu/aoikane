package phrase

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseRuleMatchesNative(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "private", "phrase-rule-reference.json"))
	if err != nil {
		t.Skip("native reference is not available")
	}
	var cases []struct {
		Accent, Connection  string
		Class               int
		Moras               int32
		Rule, First, Second int
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	bad := 0
	for i, c := range cases {
		rule, first, second := ParseRule(c.Accent, c.Connection, c.Class, c.Moras)
		// native outputs that stayed -9999 are reported as the sentinel
		if rule != c.Rule || first != c.First || second != c.Second {
			bad++
			if bad <= 15 {
				t.Errorf("case %d accent=%q conn=%q class=%d moras=%d: got (%d,%d,%d) want (%d,%d,%d)", i, c.Accent, c.Connection, c.Class, c.Moras, rule, first, second, c.Rule, c.First, c.Second)
			}
		}
	}
	t.Logf("%d cases, %d mismatches", len(cases), bad)
}
