package portable

import (
	"os"
	"testing"
)

func TestFractionZeroRangeReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "fraction-zero-unit-32", 32, "d05adcceeb4f41ae094d574d647a58bf4ea5750ddad04cb10c31b415b22802ee")
	testMixedReiReadingOriginal(t, "fraction-zero-range-32", 32, "12dd1bdc50c62ef147d26945539c811f6c12ba49d5416da95bbcfc4cc4489e89")
	testMixedReiReadingOriginal(t, "range-decimal-regression-16", 16, "b468947d666033229457c849f603d06b0d21c2314232aa219e71d0a3f2f6a9e6")
}
func TestFractionZeroRangeReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_FRACTION_ZERO_RANGE_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_FRACTION_ZERO_RANGE_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/fraction_zero_range_wave_original.json", 12, 6)
}
