package portable

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"testing"
)

// These ten inputs exercise interior Arabic coefficients and the distinct
// trailing-unit expansion. The complete 32-case regression below retains every input
// rather than selecting cases by current reading equality.
func TestInteriorCoefficientReadingOriginal(t *testing.T) {
	e, err := Open(Options{AssetsDir: "../../assets"})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	for _, short := range []string{"akane", "aoi"} {
		raw, err := os.ReadFile(fmt.Sprintf("testdata/reading-original/residual-numeral-32-%s-original.json", short))
		if err != nil {
			t.Fatal(err)
		}
		var ref struct {
			Voice    string
			Complete bool
			Corpus   string `json:"corpus_sha256"`
			Cases    []struct {
				Index    int
				Text     string
				Readings []string
			}
		}
		if err := json.Unmarshal(raw, &ref); err != nil {
			t.Fatal(err)
		}
		wantVoice := VoiceAkane
		if short == "aoi" {
			wantVoice = VoiceAoi
		}
		if !ref.Complete || ref.Voice != wantVoice || len(ref.Cases) != 32 || ref.Corpus != "9ca8db9159da623a09ddaa35cd6f7f3f7c8a24739f611db29375025751b4f448" {
			t.Fatal("incomplete original coefficient capture")
		}
		for _, i := range []int{0, 1, 2, 3, 4, 5, 6, 7, 12, 28} {
			c := ref.Cases[i]
			if c.Index != i || c.Text == "" || len(c.Readings) == 0 {
				t.Fatal("original case ordering differs")
			}
			got, err := e.ReadingVoice(c.Text, ref.Voice)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, c.Readings) {
				t.Errorf("%s case %d %q: got %q; want %q", ref.Voice, i, c.Text, got, c.Readings)
			}
		}
	}
}

func TestResidualNumeralReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "residual-numeral-32", 32, "9ca8db9159da623a09ddaa35cd6f7f3f7c8a24739f611db29375025751b4f448")
}

func TestInteriorCoefficientReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_INTERIOR_COEFFICIENT_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_INTERIOR_COEFFICIENT_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/interior_coefficient_wave_original.json", 6)
}
