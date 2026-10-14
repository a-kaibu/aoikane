package portable

import (
	"os"
	"testing"
)

func TestSpaceTailReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "space-tail-32", 32, "b9398dca34446dedf764820dc88f885132c5a7bf0c46b97f5a4cb25af81096b9")
}

func TestSpaceTailReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_SPACE_TAIL_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_SPACE_TAIL_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/space_tail_wave_original.json", 6)
}
