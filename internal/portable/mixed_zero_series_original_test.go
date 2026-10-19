package portable

import (
	"os"
	"testing"
)

func TestMixedZeroSeriesReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "mixed-zero-series-33", 33, "d1af9fe05cbaf8cbef1fb424db30e5801c3c426eefaae99e5fcce7a19cf63855")
	testMixedReiReadingOriginal(t, "paired-series-script-33", 33, "3073f94d36ee22dc10b3e0278db43d97b816465412c8fd9f9258be0f77542cc3")
	testMixedReiReadingOriginal(t, "random-numeral-seed15-long", 128, "256aaad50345e865c2dd86bd7f3684489b11f67697b2aadacf9bdda29a8a9e49")
}
func TestMixedZeroSeriesReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_MIXED_ZERO_SERIES_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_MIXED_ZERO_SERIES_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/mixed_zero_series_wave_original.json", 10, 8)
}
