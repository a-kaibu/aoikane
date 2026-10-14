package portable

import (
	"os"
	"testing"
)

func TestPointUnitPairsReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "point-unit-pairs-24", 24, "1f2fcc1ec49572e2e493d5ea207b2c18eb24088b71d9b0bf776ac93b7de19613")
	testMixedReiReadingOriginal(t, "point-unit-pairs-32", 32, "702265493dbb36bfe9097877b9a71f105c46ed026c6ad83728a5c1df795755d0")
}

func TestPointUnitPairsReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_POINT_UNIT_PAIRS_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_POINT_UNIT_PAIRS_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/point_unit_pairs_wave_original.json", 8)
}
