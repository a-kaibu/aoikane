package phrase

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/a-kaibu/aoikane/internal/cabocha"
	"github.com/a-kaibu/aoikane/internal/frontend"
)

// Start from the original text, never from captured tokens, features, or spans.
func TestCabochaTextOriginal(t *testing.T) {
	f := openFrontend(t)
	analyzer, err := frontend.OpenAnalyzer(filepath.Join("..", "..", "private", "frontend-reconstruction", "frontend-data", "kansai", "lang", "smecab"))
	if os.IsNotExist(err) {
		t.Skip("private Kansai dictionary unavailable")
	}
	if err != nil {
		t.Fatal(err)
	}
	f.Analyzer = analyzer
	svm, err := cabocha.LoadSVM(filepath.Join("..", "..", "private", "cabocha", "standard-dep.ipa.model"))
	if os.IsNotExist(err) {
		t.Skip("private dependency model unavailable")
	}
	if err != nil {
		t.Fatal(err)
	}
	f.Dependency = svm
	for _, tc := range []struct{ name, hash string }{
		{"cabocha-trees-akane", "398f98018c19c0614657d273a006eb50734f77147fd95649a672daa34ad7b6f2"},
		{"cabocha-trees-corpus", "604152491111ce1112d270f7da81980f14a8c3562f4774549488333ee5ba33e9"},
		{"cabocha-trees-brackets", "e5f2b5e4036b1d0e9f952d33e80cc8b89c65729c146c4dbc28218f9484fa27e5"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, err := os.ReadFile(filepath.Join("..", "..", "private", tc.name, "svm-talk.json"))
			if os.IsNotExist(err) {
				t.Skip("private original CaboCha text reference unavailable")
			}
			if err != nil {
				t.Fatal(err)
			}
			if fmt.Sprintf("%x", sha256.Sum256(b)) != tc.hash {
				t.Fatal("original reference hash differs")
			}
			type originalChunk struct {
				Start, Size, Head, Func, Link int
				Features                      []string
				ScoreBits                     uint32 `json:"score_bits"`
			}
			var ref struct {
				Text  string
				Trees []struct {
					Tokens []struct {
						Surface, Normalized string
						Features            []string
					}
					Chunks, Result []originalChunk
				}
			}
			if err := json.Unmarshal(b, &ref); err != nil {
				t.Fatal(err)
			}
			decode := func(value string) string {
				raw, err := hex.DecodeString(value)
				if err != nil {
					t.Fatal(err)
				}
				text, err := f.Codec.Decode(raw)
				if err != nil {
					t.Fatal(err)
				}
				return text
			}
			decodeAll := func(values []string) []string {
				result := make([]string, len(values))
				for i, v := range values {
					result[i] = decode(v)
				}
				return result
			}
			sentences := SplitSentences(ref.Text)
			if len(sentences) != len(ref.Trees) {
				t.Fatalf("sentence count %d, want %d", len(sentences), len(ref.Trees))
			}
			totalTokens, totalChunks := 0, 0
			for i, text := range sentences {
				normalized, err := frontend.NormalizeText(cp932Compatible.Replace(text))
				if err != nil {
					t.Fatal(err)
				}
				words, err := f.Analyzer.Analyze(normalized)
				if err != nil {
					t.Fatal(err)
				}
				words, generated, err := f.groupWords(words)
				if err != nil {
					t.Fatal(err)
				}
				input, err := f.cabochaInputGenerated(words, generated)
				if err != nil {
					t.Fatal(err)
				}
				tree := ref.Trees[i]
				if len(input.Tokens) != len(tree.Tokens) {
					t.Fatalf("%q: %d tokens, want %d", text, len(input.Tokens), len(tree.Tokens))
				}
				for k, got := range input.Tokens {
					want := tree.Tokens[k]
					fields := decodeAll(want.Features)
					if len(fields) < 6 {
						t.Fatal("short original IPA fields")
					}
					if got.Surface != decode(want.Surface) || got.Normalized != decode(want.Normalized) || !slices.Equal(got.Features, fields[:6]) {
						t.Fatalf("%q token %d %q: got %v, want %v", text, k, got.Surface, got.Features, fields[:6])
					}
				}
				spans := input.spans(f.Chunker)
				selected, err := cabocha.SelectIPA(input.Tokens, spans)
				if err != nil {
					t.Fatal(err)
				}
				if len(selected) != len(tree.Chunks) {
					t.Fatalf("%q: %d chunks, want %d", text, len(selected), len(tree.Chunks))
				}
				for k, got := range selected {
					want := tree.Chunks[k]
					if got.Start != want.Start || got.Size != want.Size || got.Head != want.Head || got.Func != want.Func || !slices.Equal(got.Features, decodeAll(want.Features)) {
						t.Fatalf("%q chunk %d: selection differs", text, k)
					}
				}
				deps, err := svm.Dependencies(selected, f.Codec)
				if err != nil {
					t.Fatal(err)
				}
				for k, got := range deps {
					want := tree.Result[k]
					if got.Link != want.Link || math.Float32bits(got.Score) != want.ScoreBits {
						t.Fatalf("%q chunk %d: dependency %+v (%08x), want %d (%08x)", text, k, got, math.Float32bits(got.Score), want.Link, want.ScoreBits)
					}
				}
				doc, err := f.Graph(text)
				if err != nil {
					t.Fatal(err)
				}
				if err := doc.attachSingleSentence(); err != nil {
					t.Fatal(err)
				}
				chunk := doc.Sentences[0].Chunk
				for k, want := range tree.Result {
					if chunk == nil || chunk.First == nil || chunk.First == doc.EOS {
						t.Fatalf("%q: missing graph chunk %d", text, k)
					}
					if chunk.F28 != int32(k) || chunk.F34 != int32(want.Link) || chunk.First.Surface != decode(tree.Tokens[want.Start].Surface) {
						t.Fatalf("%q graph chunk %d: index %d link %d first %q differs", text, k, chunk.F28, chunk.F34, chunk.First.Surface)
					}
					chunk = chunk.Next
				}
				if chunk == nil || chunk.First != doc.EOS {
					t.Fatalf("%q: extra graph chunks", text)
				}
				totalTokens += len(input.Tokens)
				totalChunks += len(selected)
			}
			t.Logf("%d text sentences -> %d tokens (six IPA fields) -> %d chunks -> all Selector features, dependency links and scores match original", len(sentences), totalTokens, totalChunks)
		})
	}
}

// Compare the actual AIChunk graph immediately after CaboCha. BANCHA can
// subsequently insert S chunks; those are a separate, still missing pass.
func TestCabochaInputGraphOriginal(t *testing.T) {
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
	cases := aoiCases(t, "stages_akane")
	if len(cases) == 0 {
		t.Skip("private original Kansai graph references unavailable")
	}
	if len(cases) != 79 {
		t.Fatalf("incomplete native graph references: %d, want 79", len(cases))
	}
	chunks := 0
	for _, c := range cases {
		want, err := c.Enter.Load(f.Codec)
		if err != nil {
			t.Fatal(err)
		}
		normalized, err := frontend.NormalizeText(cp932Compatible.Replace(c.Text))
		if err != nil {
			t.Fatal(err)
		}
		words, err := f.Analyzer.Analyze(normalized)
		if err != nil {
			t.Fatal(err)
		}
		words, generated, err := f.groupWords(words)
		if err != nil {
			t.Fatal(err)
		}
		input, err := f.cabochaInputGenerated(words, generated)
		if err != nil {
			t.Fatal(err)
		}
		spans := input.spans(f.Chunker)
		selection, err := cabocha.SelectIPA(input.Tokens, spans)
		if err != nil {
			t.Fatal(err)
		}
		deps, err := f.Dependency.Dependencies(selection, f.Codec)
		if err != nil {
			t.Fatal(err)
		}
		chunk := want.Sentences[0].Chunk
		for i, span := range spans {
			if chunk == nil || chunk.First == want.EOS {
				t.Fatalf("%q: missing original chunk %d", c.Text, i)
			}
			wantFirst, err := f.Codec.Encode(chunk.First.Surface)
			if err != nil {
				t.Fatal(err)
			}
			gotFirst, err := f.Codec.Encode(input.Tokens[span.Start].Surface)
			if err != nil {
				t.Fatal(err)
			}
			if string(gotFirst) != string(wantFirst) || chunk.F28 != int32(i) || chunk.F2c != -1 || chunk.F30 != 0x44 || chunk.F34 != int32(deps[i].Link) {
				t.Errorf("%q chunk %d: got %q link %d, want %q %d/%d/%x/%d", c.Text, i, input.Tokens[span.Start].Surface, deps[i].Link, chunk.First.Surface, chunk.F28, chunk.F2c, chunk.F30, chunk.F34)
			}
			chunk = chunk.Next
			chunks++
		}
		if chunk == nil || chunk.First != want.EOS {
			t.Fatalf("%q: extra original chunks", c.Text)
		}
	}
	t.Logf("%d original Kansai texts -> pre-BANCHA AIChunk graphs, %d chunk records compared", len(cases), chunks)
}
