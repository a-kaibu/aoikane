package portable

import (
	"os"
	"testing"
)

func TestNewlineEndReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_NEWLINE_END_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_NEWLINE_END_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/newline_end_wave_original.json", 4)
}
