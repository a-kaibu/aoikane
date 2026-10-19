package portable

import (
	"os"
	"testing"
)

func TestUnitZeroTableBoundaryReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "fractional-counter-key-33", 33, "ead280f43aab956f97c7a5593c998ab6c33581045b692377fa1f793a3c19f8d2")
	testMixedReiReadingOriginal(t, "unit-zero-table-boundary-33", 33, "44f6c6411c436e36f96425eb7e295d84ae9656430156720b6e05e0b335676d28")
	testMixedReiReadingOriginal(t, "random-numeral-seed14-long", 128, "c1d97e81694cbc700afbb1d96424ae5f5cede5cfa4289fea89753eb5567cfab7")
}

func TestUnitZeroTableBoundaryReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_UNIT_ZERO_TABLE_BOUNDARY_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_UNIT_ZERO_TABLE_BOUNDARY_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/unit_zero_table_boundary_wave_original.json", 12, 6)
}
