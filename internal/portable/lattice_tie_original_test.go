package portable

import (
	"github.com/a-kaibu/aoikane/internal/engine"
	"github.com/a-kaibu/aoikane/internal/phrase"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"testing"
)

func TestLatticeTieReadingOriginal(t *testing.T) {
	raw, err := os.ReadFile("testdata/lattice_tie_reading_original.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		DLL      string   `json:"dll_sha256"`
		Captures []string `json:"capture_sha256"`
		Cases    []struct {
			Text, Voice string
			Readings    []string
		}
	}
	if err = json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.DLL != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(ref.Captures) != 2 || len(ref.Cases) != 64 {
		t.Fatal("original lattice tie provenance differs")
	}
	e, err := Open(Options{AssetsDir: "../../assets"})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	for _, c := range ref.Cases {
		got, err := e.ReadingVoice(c.Text, c.Voice)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(got, c.Readings) {
			t.Errorf("%s %q: got %q; want %q", c.Voice, c.Text, got, c.Readings)
		}
	}
}

func TestLatticeTieMorphologyOriginal(t *testing.T) {
	raw, err := os.ReadFile("testdata/lattice_tie_morphology_original.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		DLL      string `json:"dll_sha256"`
		Snapshot string `json:"snapshot_sha256"`
		Input    string
		Sentence int
		Words    []struct {
			Surface, Base string
			POS           uint32
			Candidate     phrase.Candidate
		}
	}
	if err := json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.DLL != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(ref.Snapshot) != 64 || ref.Sentence != 1 || len(ref.Words) != 7 {
		t.Fatal("native morphology reference differs")
	}
	inputs := phrase.SplitSentences(ref.Input)
	if len(inputs) != 2 {
		t.Fatal("native sentence count differs")
	}
	e, err := Open(Options{AssetsDir: "../../assets"})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	doc, err := e.frontend.Graph(inputs[ref.Sentence])
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Morphs) != len(ref.Words)+1 {
		t.Fatalf("morph count %d; want %d plus EOS", len(doc.Morphs), len(ref.Words))
	}
	for i, w := range ref.Words {
		m := doc.Morphs[i]
		c := m.SelectedCandidate()
		if m.Surface != w.Surface || m.Base != w.Base || m.POS != w.POS || c == nil || *c != w.Candidate {
			t.Errorf("word %d: %q/%q/%x/%+v; native %q/%q/%x/%+v", i, m.Surface, m.Base, m.POS, c, w.Surface, w.Base, w.POS, w.Candidate)
		}
	}
}

func TestLatticeTieReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_LATTICE_TIE_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_LATTICE_TIE_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	raw, err := os.ReadFile("testdata/lattice_tie_wave_original.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		CPU     string
		DLL     string   `json:"dll_sha256"`
		Reports []string `json:"report_sha256"`
		Threads int      `json:"original_threads"`
		AVX512  bool     `json:"avx512_convolutions"`
		Cases   []struct {
			Text, Voice string
			Bytes       int
			SHA256      string
		}
	}
	if err = json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.DLL != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(ref.Reports) != 2 || ref.CPU != "AMD64 Family 25 Model 33 Stepping 0, AuthenticAMD" || ref.Threads != 6 || ref.AVX512 || len(ref.Cases) != 6 {
		t.Fatal("ordinary original WAV profile differs")
	}
	e, err := Open(Options{AssetsDir: "../../assets", OriginalThreads: ref.Threads})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	for i, c := range ref.Cases {
		if (c.Voice != VoiceAkane && c.Voice != VoiceAoi) || c.Bytes < 44 || len(c.SHA256) != 64 {
			t.Fatal("original WAV reference is incomplete")
		}
		t.Run(fmt.Sprintf("%s/case-%d", c.Voice, i), func(t *testing.T) {
			got, err := e.Synthesize(context.Background(), engine.Request{Text: c.Text, Voice: c.Voice})
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != c.Bytes || fmt.Sprintf("%x", sha256.Sum256(got)) != c.SHA256 {
				t.Fatalf("whole original WAV differs: bytes=%d SHA256=%x; want %d %s", len(got), sha256.Sum256(got), c.Bytes, c.SHA256)
			}
			t.Logf("whole original WAV identical: %q, %d bytes", c.Text, len(got))
		})
	}
}
