//go:build (windows || darwin || linux) && (amd64 || arm64)

package portable

import (
	"github.com/a-kaibu/aoikane/internal/engine"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

type controlledCase struct {
	Name    string         `json:"name"`
	Request engine.Request `json:"request"`
	Bytes   int            `json:"bytes,omitempty"`
	SHA256  string         `json:"sha256,omitempty"`
	Equal   bool           `json:"equal,omitempty"`
}

type controlledReference struct {
	Source         string           `json:"source"`
	CPU            string           `json:"cpu"`
	Threads        int              `json:"original_threads"`
	CorpusHash     string           `json:"corpus_sha256"`
	CorpusFileHash string           `json:"corpus_file_sha256"`
	DLLHash        string           `json:"dll_sha256"`
	TorchHash      string           `json:"torch_dll_sha256"`
	Cases          []controlledCase `json:"cases"`
}

// Git checkout line endings do not change the requested settings.
func controlledCorpusHash(raw []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))))
}

// The original requests and WAV digests are retained for strict replay without
// the licensed DLL. This is opt-in because complete controlled synthesis is slow.
func TestAkaneControlledReferenceWAV(t *testing.T) {
	testControlledReferenceWAV(t, "testdata/akane_controlled_original.json", "../../tools/corpus/akane-controlled-requests.json")
}
func TestControlledBoundaryReferenceWAV(t *testing.T) {
	testControlledReferenceWAV(t, "testdata/controlled_boundary_original.json", "../../tools/corpus/controlled-boundary-requests.json")
}
func testControlledReferenceWAV(t *testing.T, referenceFile, corpusFile string) {
	if os.Getenv("AOIKANE_CONTROLLED_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_CONTROLLED_REFERENCE_AUDIT=1 for strict original controlled WAV replay")
	}
	raw, err := os.ReadFile(referenceFile)
	if err != nil {
		t.Fatal(err)
	}
	var ref controlledReference
	if err = json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.CPU != "AMD64 Family 25 Model 33 Stepping 0, AuthenticAMD" || ref.Threads != 6 || ref.DLLHash != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || ref.TorchHash != "00ca5c937940a94d052dbe52ec5fb0909db11ffbfccfd6e65e19b468169ff5e5" || len(ref.Cases) != 8 {
		t.Fatal("ordinary original controlled reference provenance differs")
	}
	corpus, err := os.ReadFile(corpusFile)
	if err != nil {
		t.Fatal(err)
	}
	if controlledCorpusHash(corpus) != ref.CorpusHash {
		t.Fatal("controlled request corpus differs from original")
	}
	var requests []controlledCase
	if err = json.Unmarshal(corpus, &requests); err != nil {
		t.Fatal(err)
	}
	if len(requests) != len(ref.Cases) {
		t.Fatal("controlled request count differs")
	}
	e, err := Open(Options{AssetsDir: "../../assets", OriginalThreads: ref.Threads})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	for i, c := range ref.Cases {
		expected := requests[i]
		expected.Request.Defaults()
		if c.Name != expected.Name || c.Request != expected.Request || (c.Request.Voice != VoiceAkane && c.Request.Voice != VoiceAoi) || !c.Equal || c.Bytes < 44 || len(c.SHA256) != 64 {
			t.Fatal("saved original controlled request differs")
		}
		if err := c.Request.Validate(); err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(t *testing.T) {
			got, err := e.Synthesize(context.Background(), c.Request)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != c.Bytes || fmt.Sprintf("%x", sha256.Sum256(got)) != c.SHA256 {
				t.Fatalf("controlled WAV differs: request=%+v bytes=%d SHA256=%x; want bytes=%d SHA256=%s", c.Request, len(got), sha256.Sum256(got), c.Bytes, c.SHA256)
			}
			t.Logf("ordinary original complete controlled WAV identical: %d bytes SHA256=%s", c.Bytes, c.SHA256)
		})
	}
}
