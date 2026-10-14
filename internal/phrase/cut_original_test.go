package phrase

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/a-kaibu/aoikane/internal/cabocha"
	"github.com/a-kaibu/aoikane/internal/frontend"
)

func originalCutRules(t *testing.T, name string) *CutRules {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "private", name+".json"))
	if os.IsNotExist(err) {
		t.Skip("private original cut-rule asset unavailable")
	}
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(b)) != "925e9f015ec3174b41186586823a3400c8317d45118d57f7794cb71f2c680dfd" {
		t.Fatal("original cut-rule asset hash differs")
	}
	r, err := ParseCutRules(strings.NewReader(string(b)))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

type cutChunkRecord struct {
	Index   int
	Surface string
	Fields  [4]int32
}

func cutChunkRecords(t *testing.T, doc *Document) []cutChunkRecord {
	t.Helper()
	var out []cutChunkRecord
	indices := map[*Morph]int{}
	for i, m := range doc.Morphs {
		indices[m] = i
	}
	var previous *Chunk
	for c := doc.Sentences[0].Chunk; c != nil; c = c.Next {
		if len(out) > 4096 || c.Prev != previous || c.First.Chunk != c {
			t.Fatal("invalid chunk links")
		}
		out = append(out, cutChunkRecord{indices[c.First], c.First.Surface, [4]int32{c.F28, c.F2c, c.F30, c.F34}})
		previous = c
	}
	return out
}
func cutRangeChunk(t *testing.T, s Snapshot, doc *Document, address string) *Chunk {
	t.Helper()
	raw, err := hex.DecodeString(s.Nodes[address].Raw)
	if err != nil || len(raw) < 0x38 {
		t.Fatal("missing cut-range chunk")
	}
	for c := doc.Sentences[0].Chunk; c != nil; c = c.Next {
		if c.F28 == i32(raw, 0x28) && c.F2c == i32(raw, 0x2c) {
			return c
		}
	}
	t.Fatal("cut-range chunk not found")
	return nil
}

func TestCutOriginal(t *testing.T) {
	rules := originalCutRules(t, "bancut-akane")
	codec := testCodec(t)
	decode := func(value string) string {
		raw, err := hex.DecodeString(value)
		if err != nil {
			t.Fatal(err)
		}
		text, err := codec.Decode(raw)
		if err != nil {
			t.Fatal(err)
		}
		return text
	}
	type originalMorph struct {
		POS        uint32 `json:"pos"`
		Form       uint32 `json:"form"`
		Type       uint32 `json:"type"`
		Connection string `json:"connection"`
	}
	morph := func(m *originalMorph) *Morph {
		if m == nil {
			return nil
		}
		return &Morph{POS: m.POS, POS2: m.Form, POS3: m.Type, Cands: []Candidate{{Connection: decode(m.Connection)}}}
	}
	decisions, ranges, inserted := 0, 0, 0
	for _, name := range []string{"bancut-akane", "bancut-akane-extra"} {
		path := filepath.Join("..", "..", "private", name+".jsonl")
		b, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			t.Skip("private original cut-stage reference unavailable")
		}
		if err != nil {
			t.Fatal(err)
		}
		hashes := map[string]string{
			"bancut-akane":       "688ed020f1095f172cd5a445cfd135ab5b8e19ad8e68906b129ae64f9aa6d71a",
			"bancut-akane-extra": "a059718650b96a27b00b7590a448351c000f4e26fc020824db254d0368219943",
		}
		if fmt.Sprintf("%x", sha256.Sum256(b)) != hashes[name] {
			t.Fatal("original cut-stage capture hash differs")
		}
		dec := json.NewDecoder(strings.NewReader(string(b)))
		var before *Document
		for {
			var record struct {
				Snapshot
				Prev, Current, Next *originalMorph
				Matches             []bool
				Result              bool
				Error               string
			}
			if err := dec.Decode(&record); err == io.EOF {
				break
			} else if err != nil {
				t.Fatal(err)
			}
			switch record.Event {
			case "error":
				t.Fatal(record.Error)
			case "cut_decision":
				p, c, n := morph(record.Prev), morph(record.Current), morph(record.Next)
				if len(record.Matches) != len(rules.Judges) {
					t.Fatal("incomplete original judge observations")
				}
				for i, j := range rules.Judges {
					if got := rules.judgeMatches(j, p, c, n); got != record.Matches[i] {
						t.Fatalf("%s %q decision %d judge %d: %v, want %v", name, record.Input, decisions, i, got, record.Matches[i])
					}
				}
				if got := rules.shouldCut(p, c, n); got != record.Result {
					t.Fatalf("%q decision %d: %v, want %v", record.Input, decisions, got, record.Result)
				}
				if record.Result {
					inserted++
				}
				decisions++
			case "stage_enter_115110":
				if before != nil {
					t.Fatal("nested cut-stage observation")
				}
				before, err = record.Snapshot.Load(codec)
				if err != nil {
					t.Fatal(err)
				}
				begin, end := cutRangeChunk(t, record.Snapshot, before, record.CutBegin), cutRangeChunk(t, record.Snapshot, before, record.CutEnd)
				if err := rules.split(begin, end); err != nil {
					t.Fatal(err)
				}
			case "stage_leave_115110":
				if before == nil {
					t.Fatal("missing original cut-stage input")
				}
				after, err := record.Snapshot.Load(codec)
				if err != nil {
					t.Fatal(err)
				}
				got, want := cutChunkRecords(t, before), cutChunkRecords(t, after)
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("%q cut range:\ngot  %+v\nwant %+v", record.Input, got, want)
				}
				before = nil
				ranges++
			}
		}
		if before != nil {
			t.Fatal("missing original cut-stage output")
		}
	}
	if decisions != 131 || ranges != 54 || inserted != 8 {
		t.Fatalf("incomplete original cut comparisons: %d decisions, %d ranges, %d cuts", decisions, ranges, inserted)
	}
	t.Logf("%d original decisions, %d individual judge results, %d ranges, %d inserted S chunks: identical", decisions, decisions*len(rules.Judges), ranges, inserted)
}

func TestCutRulesValidation(t *testing.T) {
	original := originalCutRules(t, "bancut-akane")
	for _, change := range []func(*CutRules){
		func(r *CutRules) { r.Version++ },
		func(r *CutRules) { r.SourceSHA256 = "unknown" },
		func(r *CutRules) { r.Groups["cycle"] = []CutCondition{{Kind: "J", Value: "cycle"}} },
		func(r *CutRules) { r.Groups["unknown"] = []CutCondition{{Kind: "J", Value: "missing"}} },
		func(r *CutRules) { r.Groups["invalid"] = []CutCondition{{Kind: "?"}} },
		func(r *CutRules) { r.Judges[0].Clauses[0].Position = "?" },
		func(r *CutRules) { r.Judges[0].Clauses[0].Group = "missing" },
	} {
		b, _ := json.Marshal(original)
		var r CutRules
		json.Unmarshal(b, &r)
		change(&r)
		b, _ = json.Marshal(r)
		if _, err := ParseCutRules(strings.NewReader(string(b))); err == nil {
			t.Fatal("accepted invalid rules")
		}
	}
	for _, suffix := range []string{"{}", "garbage"} {
		b, _ := json.Marshal(original)
		if _, err := ParseCutRules(strings.NewReader(fmt.Sprintf("%s%s", b, suffix))); err == nil {
			t.Fatal("accepted trailing data")
		}
	}
}

// Use the original post-BANCHA words to isolate chunk splitting from numeral
// and reading selection. Remove only S chunks, then recompute their full graph.
func TestCutGraphsOriginal(t *testing.T) {
	r := originalCutRules(t, "bancut-akane")
	codec := testCodec(t)
	cases := aoiCases(t, "stages_akane")
	if len(cases) == 0 {
		t.Skip("private original Kansai graphs unavailable")
	}
	if len(cases) != 79 {
		t.Fatalf("incomplete original graph references: %d", len(cases))
	}
	total, added := 0, 0
	for _, tc := range cases {
		doc, err := tc.Leave.Load(codec)
		if err != nil {
			t.Fatal(err)
		}
		want := cutChunkRecords(t, doc)
		for c := doc.Sentences[0].Chunk; c != nil; {
			next := c.Next
			if c.F30 == 'S' {
				c.First.Chunk = nil
				c.Prev.Next = c.Next
				if c.Next != nil {
					c.Next.Prev = c.Prev
				}
				added++
			}
			c = next
		}
		if err := doc.ApplyCut(r); err != nil {
			t.Fatalf("%q: %v", tc.Text, err)
		}
		if got := cutChunkRecords(t, doc); !reflect.DeepEqual(got, want) {
			t.Fatalf("%q:\ngot %+v\nwant %+v", tc.Text, got, want)
		}
		total += len(want)
	}
	if added != 34 || total != 385 {
		t.Fatalf("incomplete original graphs: %d records, %d S chunks", total, added)
	}
	t.Logf("%d original Kansai word graphs, %d chunk records, %d S chunks reconstructed identically", len(cases), total, added)
}

// Start from text and run the Go analyzer, reading selection, CaboCha and cut
// passes. This complements the isolated stage tests above.
func TestCutTextOriginal(t *testing.T) {
	f := openFrontend(t)
	var err error
	f.Analyzer, err = frontend.OpenAnalyzer(filepath.Join("..", "..", "private", "frontend-reconstruction", "frontend-data", "kansai", "lang", "smecab"))
	if os.IsNotExist(err) {
		t.Skip("private Kansai dictionary unavailable")
	}
	if err != nil {
		t.Fatal(err)
	}
	f.Dependency, err = cabocha.LoadSVM(filepath.Join("..", "..", "private", "cabocha", "standard-dep.ipa.model"))
	if os.IsNotExist(err) {
		t.Skip("private dependency model unavailable")
	}
	if err != nil {
		t.Fatal(err)
	}
	f.Cut = originalCutRules(t, "bancut-akane")
	total := 0
	for _, name := range []string{"bancut-akane", "bancut-akane-extra"} {
		snapshots, err := ReadSnapshots(filepath.Join("..", "..", "private", name+".jsonl"))
		if os.IsNotExist(err) {
			t.Skip("private original cut-stage references unavailable")
		}
		if err != nil {
			t.Fatal(err)
		}
		final := map[string]Snapshot{}
		for _, s := range snapshots {
			if s.Event == "stage_leave_115110" {
				final[s.Input] = s
			}
		}
		for text, s := range final {
			want, err := s.Load(f.Codec)
			if err != nil {
				t.Fatal(err)
			}
			got, err := f.Graph(text)
			if err != nil {
				t.Fatalf("%q: %v", text, err)
			}
			if err := got.attachSingleSentence(); err != nil {
				t.Fatal(err)
			}
			if err := got.ApplyCut(f.Cut); err != nil {
				t.Fatal(err)
			}
			a, b := cutChunkRecords(t, got), cutChunkRecords(t, want)
			if !reflect.DeepEqual(a, b) {
				t.Fatalf("%q:\ngot %+v\nwant %+v", text, a, b)
			}
			total++
		}
	}
	if total != 11 {
		t.Fatalf("incomplete original text references: %d, want 11", total)
	}
	t.Logf("%d original texts -> Go-only BANCHA chunk graphs: identical", total)
}
