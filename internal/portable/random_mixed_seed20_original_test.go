package portable

import (
	"os"
	"testing"
)

func TestRandomMixedSeed20ReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "random-mixed-seed20-long", 128, "1feafd104c5acb073b6d3dc75d4e660238dfa6b7e3b3bc33a70dbc78dbb846f5")
}

func TestRandomMixedSeed20ReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_MIXED_SEED20_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_MIXED_SEED20_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/mixed_seed20_wave_original.json", 8, 2)
}
