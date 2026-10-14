package portable

import (
	"os"
	"testing"
)

func TestBareUnitRangeReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "bare-unit-range-20", 20, "2e5b568d3edad611fbff6c38f2e48b007c74f78bce2fca28e01c82f221f9b130")
}

func TestBareUnitRangeBoundaryReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "bare-unit-range-boundary-32", 32, "d811dbe147d3307996138648190e9f6cd9c5f5f80e6807a83e0233356c933c68")
}

func TestBareUnitRangeReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_BARE_UNIT_RANGE_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_BARE_UNIT_RANGE_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/bare_unit_range_wave_original.json", 8, 4)
}
