package portable

import (
	"os"
	"testing"
)

func TestSecondsBracketReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "seconds-bracket-32", 32, "71997e2574f8a067572a89fe79b4a7be82a153d6893a19b70b72a113741f995a")
	testMixedReiReadingOriginal(t, "random-numeral-seed10-long", 128, "99808368c46a835a5777f75ccb8b1c492e416b42ab14b46bd3acd1ac00fbe271")
}
func TestSecondsBracketReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_SECONDS_BRACKET_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_SECONDS_BRACKET_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/seconds_bracket_wave_original.json", 12, 4)
}
