package portable

import (
	"os"
	"testing"
)

func TestCounterFlatUnitReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "counter-flat-unit-32", 32, "142df004e0fbd07022e48d70c954de24788ce26a6ed00dcb57ee0b81354e7415")
	testMixedReiReadingOriginal(t, "counter-flat-c2-12", 12, "391753f59be5879c5a5da12e53c1d5f7fc9f28808a0a2fbe813aa1b537ae2446")
}

func TestCounterFlatUnitReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_COUNTER_FLAT_UNIT_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_COUNTER_FLAT_UNIT_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/counter_flat_unit_wave_original.json", 8)
}
