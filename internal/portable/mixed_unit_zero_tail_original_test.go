package portable

import (
	"os"
	"testing"
)

func TestMixedUnitZeroTailReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "seconds-large-prefix-33", 33, "9e0eade92bd8e99e28d90a5c6540d79f5df577874e07cf2b57184c77d40d51d2")
	testMixedReiReadingOriginal(t, "mixed-unit-zero-tail-33", 33, "9f2485560359d48042e4da9b03efcf5b9e3c8e83cdc27415ecb9a164ac9acd71")
	testMixedReiReadingOriginal(t, "random-numeral-seed13-long", 128, "1765487c8544483d3fcc5efa3f20df3fbb0f91c7bf13179e25da0e9234bb6b97")
}

func TestMixedUnitZeroTailReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_MIXED_UNIT_ZERO_TAIL_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_MIXED_UNIT_ZERO_TAIL_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/mixed_unit_zero_tail_wave_original.json", 12, 6)
}
