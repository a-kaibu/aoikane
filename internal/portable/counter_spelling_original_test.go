package portable

import (
	"os"
	"testing"
)

func TestCounterSpellingReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "counter-yen-context-33", 33, "18f26ff426ba0c80dc859fa8c1ee28c044d4050e2205c55f45bab400b72b79bd")
	testMixedReiReadingOriginal(t, "counter-spelling-bang-33", 33, "bd38952354ee193c8e6f9c895110b2f0d80dc60a8e4fe058ed255422c5d2f9aa")
	testMixedReiReadingOriginal(t, "random-numeral-seed16-long", 128, "09a72355c0f8547f14252d3536b938bbbd681583b0d529c7d6dc24b02879a5ce")
	testMixedReiReadingOriginal(t, "random-numeral-seed17-long", 128, "42df09208553521ceb403c50c359d21b67c926837db5dc3e1376454952f0bc12")
}
func TestCounterSpellingReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_COUNTER_SPELLING_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_COUNTER_SPELLING_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/counter_spelling_wave_original.json", 8, 4)
}
