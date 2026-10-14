package cabocha

import (
	"github.com/a-kaibu/aoikane/internal/frontend"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestDependenciesOriginal(t *testing.T) {
	for _, tc := range []struct {
		name, hash, wav string
		trees, calls    int
	}{
		{"cabocha-trees-akane", "398f98018c19c0614657d273a006eb50734f77147fd95649a672daa34ad7b6f2", "66ad8e91dee4a8c12e610a369db335b26e9a1fcd388fb38fb68a97e76fc1399a", 2, 5},
		{"cabocha-trees-corpus", "604152491111ce1112d270f7da81980f14a8c3562f4774549488333ee5ba33e9", "22642c2b0c886a206e1f716abff480fe1e3a3eb4849bb5329d4f1fb8914e5e7e", 6, 35},
		{"cabocha-trees-brackets", "e5f2b5e4036b1d0e9f952d33e80cc8b89c65729c146c4dbc28218f9484fa27e5", "7adafc0bb7b64ac042b81532242e921cecc054d8b7457b4ea63479c122c273b3", 3, 12},
	} {
		t.Run(tc.name, func(t *testing.T) { testDependenciesOriginal(t, tc.name, tc.hash, tc.wav, tc.trees, tc.calls) })
	}
}

func testDependenciesOriginal(t *testing.T, name, hash, wav string, wantTrees, wantCalls int) {
	read := func(name, hash string) []byte {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join("..", "..", "private", filepath.FromSlash(name)))
		if os.IsNotExist(err) {
			t.Skip("private original dependency reference unavailable")
		}
		if err != nil {
			t.Fatal(err)
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != hash {
			t.Fatalf("%s hash %s", name, got)
		}
		return raw
	}
	m, err := ParseSVM(read("cabocha/standard-dep.ipa.model", "2380fc689e2b21f063869e5e1f8ae304b2d2bf5f75cad6a7421c9fa0dbd7cb4f"))
	if err != nil {
		t.Fatal(err)
	}
	codec, err := frontend.OpenCP932Codec(filepath.Join("..", "..", "assets", "data", "cp932.json"))
	if err != nil {
		t.Fatal(err)
	}
	type originalChunk struct {
		Start, Size, Head, Func, Link int
		Features                      []string
		ScoreBits                     uint32 `json:"score_bits"`
	}
	var original struct {
		Trees []struct {
			Index  int
			Tokens []struct {
				Surface, Normalized string
				Features            []string
			}
			Chunks, Result []originalChunk
		}
		Calls []struct {
			Tree, Src, Dst int
			IDs            []int
			Bits           string
		}
	}
	raw := read(name+"/svm-talk.json", hash)
	read(name+"/baseline.wav", wav)
	read(name+"/observed.wav", wav)
	if err := json.Unmarshal(raw, &original); err != nil {
		t.Fatal(err)
	}
	decode := func(value string) string {
		t.Helper()
		b, err := hex.DecodeString(value)
		if err != nil {
			t.Fatal(err)
		}
		text, err := codec.Decode(b)
		if err != nil {
			t.Fatal(err)
		}
		return text
	}
	decodeAll := func(values []string) []string {
		out := make([]string, len(values))
		for i, value := range values {
			out[i] = decode(value)
		}
		return out
	}
	candidates, features, chunks, tokens := 0, 0, 0, 0
	for _, tree := range original.Trees {
		var input []Token
		var spans []ChunkSpan
		for _, token := range tree.Tokens {
			input = append(input, Token{decode(token.Surface), decode(token.Normalized), decodeAll(token.Features)})
		}
		for _, chunk := range tree.Chunks {
			spans = append(spans, ChunkSpan{chunk.Start, chunk.Size})
		}
		selection, err := SelectIPA(input, spans)
		if err != nil {
			t.Fatal(err)
		}
		for i, got := range selection {
			want := tree.Chunks[i]
			if got.Head != want.Head || got.Func != want.Func || !slices.Equal(got.Features, decodeAll(want.Features)) {
				t.Fatalf("tree %d chunk %d: Selector differs\ngot %v\nwant %v", tree.Index, i, got.Features, decodeAll(want.Features))
			}
			features += len(got.Features)
		}
		var decisions []Candidate
		result, err := m.dependencies(selection, codec, func(c Candidate) { decisions = append(decisions, c) })
		if err != nil {
			t.Fatal(err)
		}
		for i, got := range result {
			want := tree.Result[i]
			if got.Link != want.Link || math.Float32bits(got.Score) != want.ScoreBits {
				t.Fatalf("tree %d chunk %d: dependency %+v (%08x), want %d (%08x)", tree.Index, i, got, math.Float32bits(got.Score), want.Link, want.ScoreBits)
			}
		}
		var reference []struct {
			Src, Dst int
			IDs      []int
			Bits     string
		}
		for _, call := range original.Calls {
			if call.Tree == tree.Index {
				reference = append(reference, struct {
					Src, Dst int
					IDs      []int
					Bits     string
				}{call.Src, call.Dst, call.IDs, call.Bits})
			}
		}
		if len(decisions) != len(reference) {
			t.Fatalf("tree %d: %d decisions, want %d", tree.Index, len(decisions), len(reference))
		}
		for i, got := range decisions {
			want := reference[i]
			bits, err := hex.DecodeString(want.Bits)
			if err != nil || len(bits) != 8 {
				t.Fatal("invalid score reference")
			}
			if got.Src != want.Src || got.Dst != want.Dst || !slices.Equal(got.IDs, want.IDs) || math.Float64bits(got.Score) != binary.LittleEndian.Uint64(bits) {
				t.Fatalf("tree %d decision %d: candidate %d -> %d with %d IDs differs", tree.Index, i, got.Src, got.Dst, len(got.IDs))
			}
		}
		tokens += len(input)
		chunks += len(spans)
		candidates += len(decisions)
	}
	if len(original.Trees) != wantTrees || candidates != wantCalls {
		t.Fatal("incomplete dependency observation")
	}
	t.Logf("%d trees, %d tokens, %d chunks, %d Selector features, %d candidate vectors/scores and all dependency links match original", len(original.Trees), tokens, chunks, features, candidates)
}
