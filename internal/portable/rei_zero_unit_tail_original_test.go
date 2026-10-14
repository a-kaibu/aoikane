package portable

import (
	"os"
	"testing"
)

func TestReiZeroUnitTailReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "rei-zero-unit-tail-20", 20, "09731db37783e5175e0f38ba2b257471f147aa7a5c8c704c0ea5115b65b39705")
	testMixedReiReadingOriginal(t, "rei-zero-unit-tail-boundary-8", 8, "44f6c1e6daf6dcdae79549337619ce1530079ce329dc3b9224b4de732d7dddda")
}

func TestReiZeroUnitTailReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_REI_ZERO_UNIT_TAIL_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_REI_ZERO_UNIT_TAIL_REFERENCE_AUDIT=1 for all original WAV references")
	}
	testMixedReiReferenceWAV(t, "testdata/rei_zero_unit_tail_wave_original.json", 8)
}
