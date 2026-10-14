package portable

import (
	"os"
	"testing"
)

func TestFractionCounterReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "fraction-counter-16", 16, "f4690b54764b03a55e5bc4738f836335a83368d212cafecab5a2ab367e8617db")
}
func TestFractionBoundaryReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "fraction-boundary-16", 16, "663bf8bfb2bd758414d7abdfdaf7732643d45331ead4376f8e7d17388685fd25")
}
func TestConnectorFallbackReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "connector-fallback-24", 24, "be2833a1b2008267483466c3d05f2ea34569ad1029ccecca15b261f7c3a03849")
}
func TestFractionCounterReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_FRACTION_COUNTER_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_FRACTION_COUNTER_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/fraction_counter_wave_original.json", 8, 8)
}
