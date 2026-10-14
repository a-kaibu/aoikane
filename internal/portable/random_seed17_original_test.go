package portable

import (
	"github.com/a-kaibu/aoikane/internal/difftest"
	"encoding/json"
	"os"
	"testing"
)

func TestRandomSeed17LongReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_RANDOM_SEED17_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_RANDOM_SEED17_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	raw, err := os.ReadFile("testdata/random_seed17_wave_original.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		Seed      uint64
		Generator int `json:"generator_version"`
		MaxUnits  int `json:"max_units"`
		Cases     []struct {
			Index       int
			Text, Voice string
		}
	}
	if err = json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.Seed != 17 || ref.Generator != difftest.GeneratorVersion || ref.MaxUnits != 24 || len(ref.Cases) != 8 {
		t.Fatal("original random corpus differs")
	}
	texts := difftest.Texts(17, 128, 24)
	for i, c := range ref.Cases {
		voice := VoiceAkane
		if i >= 4 {
			voice = VoiceAoi
		}
		if c.Index != i%4 || c.Text != texts[i%4] || c.Voice != voice {
			t.Fatal("original generated input/order differs")
		}
	}
	testMixedReiReferenceWAV(t, "testdata/random_seed17_wave_original.json", 8)
}
