package portable

import (
	"os"
	"testing"
)

func TestHyphenRatioReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "hyphen-ratio-20", 20, "139bdc24a550dfe4e7b5ac748fd39da19aeac74515041180e4529a0c291c3fae")
}

func TestHyphenRatioReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_HYPHEN_RATIO_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_HYPHEN_RATIO_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/hyphen_ratio_wave_original.json", 8)
}
