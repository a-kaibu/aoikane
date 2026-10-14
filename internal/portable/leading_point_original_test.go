package portable

import (
	"os"
	"testing"
)

func TestLeadingPointReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "leading-point-32", 32, "b6987cbcc6d20abd154fddc6134f40922327ffbc7bfbf019b0eece4a6dbbf606")
}

func TestLeadingPointReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_LEADING_POINT_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_LEADING_POINT_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/leading_point_wave_original.json", 8)
}
