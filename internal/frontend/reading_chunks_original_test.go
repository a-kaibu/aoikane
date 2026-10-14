package frontend

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestReadingChunksOriginal(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "private", "reading-chunk-reference.json"))
	if os.IsNotExist(err) {
		t.Skip("private original chunk reading reference unavailable")
	}
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(b)) != "3c42ddd9c346bc3319e7504dba862543058ba5c274a5272195b336960808e8cb" {
		t.Fatal("original chunk reading reference hash differs")
	}
	var ref struct {
		Source string `json:"source_sha256"`
		Cases  []struct {
			Words     []LanguageWord
			Chunks    []ReadingChunk
			Current   int
			Boundary  string
			Negate    bool
			Condition LanguageCondition
			Scope     ReadingContext
			Result    bool
		}
	}
	if err := json.Unmarshal(b, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.Source != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(ref.Cases) != 48000 {
		t.Fatal("invalid original chunk reading references")
	}
	for i, c := range ref.Cases {
		p := ReadingPredicate{Kind: "word", Condition: c.Condition}
		if c.Boundary != "" {
			p = ReadingPredicate{Kind: c.Boundary, Negate: c.Negate}
		}
		r := &ReadingRules{Conditions: map[int]ReadingPredicate{1: p}}
		got, err := r.EvaluateChunkContext(c.Scope, c.Words, c.Chunks, c.Current)
		if err != nil || got != c.Result {
			t.Fatalf("case %d: %v / %v, want %v; current=%d scope=%+v", i, got, err, c.Result, c.Current, c.Scope)
		}
	}
	t.Logf("%d P/N dependency reading scopes: identical to original", len(ref.Cases))
}
