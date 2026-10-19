package portable

import (
	"os"
	"testing"
)

func TestReiUnitPointReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "rei-unit-point-33", 33, "263f3355cf1e40220f5ee28a510b6e67e70e4bfda1af2236c3f666e706b37a4c")
}
func TestReiUnitPointReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_REI_UNIT_POINT_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_REI_UNIT_POINT_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/rei_unit_point_wave_original.json", 6, 4)
}
