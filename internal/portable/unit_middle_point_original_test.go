package portable

import (
	"os"
	"testing"
)

func TestUnitMiddlePointReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "unit-middle-point-32", 32, "e1a2f40e9e9004c5cf98a5b26331dd7a5310e4b533665c9f1818e62cdf150c55")
	testMixedReiReadingOriginal(t, "unit-table-point-16", 16, "bf95d7c807618e0254c7542219cc89b6fdd3f3a458bc9b2b2525f38d666c2c33")
}
func TestUnitMiddlePointReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_UNIT_MIDDLE_POINT_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_UNIT_MIDDLE_POINT_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/unit_middle_point_wave_original.json", 12, 4)
}
