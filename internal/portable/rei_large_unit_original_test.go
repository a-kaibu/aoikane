package portable

import (
	"os"
	"testing"
)

func TestReiLargeUnitReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "rei-large-word-16", 16, "cd9edcc7e9d150ccc98350031cab3b428f3dd03c4f8f7cd0b2016c17223152c8")
}

func TestReiLargeUnitReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_REI_LARGE_UNIT_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_REI_LARGE_UNIT_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/rei_large_unit_wave_original.json", 8)
}
