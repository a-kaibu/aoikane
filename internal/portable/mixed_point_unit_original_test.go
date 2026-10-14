package portable

import (
	"os"
	"testing"
)

func TestMixedPointUnitReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "mixed-point-unit-20", 20, "4b07a05de1aeb22771611b9b41805e3cb3ee348a327fa9034142294ff6fda508")
}

func TestMixedPointUnitReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_MIXED_POINT_UNIT_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_MIXED_POINT_UNIT_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/mixed_point_unit_wave_original.json", 6)
}
