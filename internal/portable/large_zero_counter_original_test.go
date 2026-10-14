package portable

import (
	"os"
	"testing"
)

func TestLargeZeroCounterReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "large-zero-counter-32", 32, "c04fd02a70c45c14ff418e290f1e685f237ecea7d2b62f494f9e846d5a6f7b9b")
	testMixedReiReadingOriginal(t, "large-counter-boundary-16", 16, "7fd71370e7efe062f84480f2e4262ce48fb3dee7bcf6a4e02cd2778d8f88e205")
}

func TestLargeZeroCounterReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_LARGE_ZERO_COUNTER_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_LARGE_ZERO_COUNTER_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/large_zero_counter_wave_original.json", 8)
}
