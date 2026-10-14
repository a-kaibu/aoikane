package portable

import (
	"os"
	"testing"
)

func TestZeroSmallUnitCoefficientReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "zero-small-unit-coefficient-20", 20, "993a7898690eca229d13c61266b937ebba8384270c000b1104fd7cd1e20291f5")
}

func TestZeroSmallUnitCoefficientBoundaryReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "zero-small-unit-coefficient-boundary-12", 12, "1052b89c21a7f08f8910418cf70ddc40c2e9026c179609bd2cb8c0ae983c8461")
}

func TestZeroSmallUnitCoefficientReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_ZERO_SMALL_UNIT_COEFFICIENT_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_ZERO_SMALL_UNIT_COEFFICIENT_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/zero_small_unit_coefficient_wave_original.json", 8, 4)
}
