package portable

import (
	"os"
	"testing"
)

func TestZeroCounterReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "zero-counter-context-50", 50, "9300c5c01c249f54614b61bad28858f784e51077b8c2f13c76a3137986084309")
}

func TestSecondsFractionReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "seconds-fraction-24", 24, "565c059f5131ceab634ff8974c9637934d8b96c958eef0e7f3300a13c924bc15")
}

func TestZeroCounterReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_ZERO_COUNTER_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_ZERO_COUNTER_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/zero_counter_wave_original.json", 8)
}
