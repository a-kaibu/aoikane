package portable

import (
	"os"
	"testing"
)

func TestUnitContinuationReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "unit-continuation-20", 20, "4bc0179e72c9a4c93d47cee497ec44d5bbe39709d03bca12a32f64902dba8199")
}
func TestUnitContinuationBoundaryReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "unit-continuation-boundary-12", 12, "d3981d4d0a758e737cf82c27eac16303fc0c492e1b5f488bcadfd2f18bcdbbcb")
}
func TestUnitContinuationReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_UNIT_CONTINUATION_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_UNIT_CONTINUATION_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/unit_continuation_wave_original.json", 6, 4)
}
