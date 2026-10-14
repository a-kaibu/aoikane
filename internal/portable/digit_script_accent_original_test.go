package portable

import (
	"os"
	"testing"
)

func TestDigitScriptAccentReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "digit-script-accent-16", 16, "f577169b24bf150b005b067eaa7f99e8eac0a5445d5ac35826e68bfb7598fb36")
}

// Keep all 128 pinned inputs in the normal original-reading regression.
func TestRandomNumeralSeed4ReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "random-numeral-seed4", 128, "114c1e809464c9cd337e948c826ca18a279165ccd141caf8c3c4438e9f53c61f")
}

func TestDigitScriptAccentReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_DIGIT_SCRIPT_ACCENT_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_DIGIT_SCRIPT_ACCENT_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/digit_script_accent_wave_original.json", 4)
}
