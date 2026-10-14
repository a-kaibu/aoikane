package portable

import (
	"os"
	"testing"
)

func TestPointNumeralReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "point-numeral-24", 24, "e147067680aae4248d821b86e93961729de348a527df4a5e1bc0ffe0b29bd5e4")
}

func TestPointPrefixReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "point-prefix-24", 24, "c206b345706f182bdeb25f39d408f78836cec8e9f468c6511e7f13b1fd2f122e")
}

func TestPointNumeralReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_POINT_NUMERAL_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_POINT_NUMERAL_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/point_numeral_wave_original.json", 14)
}
