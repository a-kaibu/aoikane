package portable

import (
	"os"
	"testing"
)

func TestSecondsContinuationReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "seconds-continuation-20", 20, "21028463ae06039e184c2983c1a75422ac27d9f9b9ea9891a35210b8c1d8ffa0")
}

func TestSecondsContinuationReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_SECONDS_CONTINUATION_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_SECONDS_CONTINUATION_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/seconds_continuation_wave_original.json", 10, 6)
}
