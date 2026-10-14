package portable

import (
	"github.com/a-kaibu/aoikane/internal/engine"
	"github.com/a-kaibu/aoikane/internal/phrase"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// This reference contains ordinary original Akane frontend outputs, including
// six freshly captured texts and the existing 79-sentence corpus.
func TestAkaneReadingOriginal(t *testing.T) {
	raw, err := os.ReadFile("testdata/akane_reading_original.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		DLLHash string `json:"dll_sha256"`
		Voice   string
		First   bool `json:"first_sentence_only"`
		Rules   map[string]string
		Cases   []struct{ Text, Reading string }
	}
	if err = json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.DLLHash != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || ref.Voice != VoiceAkane || !ref.First || len(ref.Cases) != 85 || len(ref.Rules) != 5 {
		t.Fatal("original Akane reference provenance differs")
	}
	for name, want := range ref.Rules {
		data, err := os.ReadFile(filepath.Join("../../assets/rules", name))
		if err != nil {
			t.Fatal(err)
		}
		if fmt.Sprintf("%x", sha256.Sum256(data)) != want {
			t.Fatal("original rule asset differs", name)
		}
	}
	e, err := Open(Options{AssetsDir: "../../assets"})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	if voices := e.Voices(); len(voices) != 2 || voices[0].ID != VoiceAoi || voices[1].ID != VoiceAkane {
		t.Fatal("both voices must be selectable")
	}
	for _, c := range ref.Cases {
		sentences := phrase.SplitSentences(c.Text)
		if len(sentences) == 0 {
			t.Fatal("empty reference")
		}
		got, err := e.ReadingVoice(sentences[0], VoiceAkane)
		if err != nil {
			t.Fatalf("%q: %v", c.Text, err)
		}
		if len(got) != 1 || got[0] != c.Reading {
			t.Errorf("%q got %q want %q", c.Text, got, c.Reading)
		}
	}
	t.Log("85 original Akane texts: formatted readings and accents identical")
}

// The digest is the complete WAV from the ordinary original CLI, without
// Frida, forced native threads, or injected intermediate model tensors.
func TestAkaneOriginalWAV(t *testing.T) {
	e, err := Open(Options{AssetsDir: "../../assets"})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	got, err := e.Synthesize(context.Background(), engine.Request{Text: "今日はええ天気やな。", Voice: VoiceAkane})
	if err != nil {
		t.Fatal(err)
	}
	const want = "be86d13c4c80e387b7622b84cc561a153c8b444db22901e7045ff681e7218766"
	if len(got) != 232426 || len(got) < 44 || binary.LittleEndian.Uint32(got[24:28]) != 48000 || fmt.Sprintf("%x", sha256.Sum256(got)) != want {
		t.Fatalf("original Akane WAV differs: bytes=%d SHA256=%x", len(got), sha256.Sum256(got))
	}
	t.Logf("Original Akane full WAV: 116191 PCM16 samples identical; SHA256 %s", want)
}
