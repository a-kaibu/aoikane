package portable

import (
	"os"
	"testing"
)

func TestSmallUnitZeroReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "small-unit-zero-prefix-20", 20, "891b6d0a55a22a8750a445bce75057a372cf18681602b81a9ae219e7e9727bb2")
	testMixedReiReadingOriginal(t, "small-unit-zero-boundary-8", 8, "0aa87216d663129fccd653386acd9cc8b5e08aacdc9675b36fd0c2b24b263ddd")
	testMixedReiReadingOriginal(t, "small-unit-zero-coefficient-8", 8, "feb206db4b7666d4bf81b20513c93d6ced6e02127fc83b68f346d9845eaabcf5")
}

func TestSmallUnitZeroReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_SMALL_UNIT_ZERO_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_SMALL_UNIT_ZERO_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/small_unit_zero_wave_original.json", 8)
}
