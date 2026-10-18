package portable

import (
	"os"
	"testing"
)

func TestUnitCounterTailReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "unit-counter-prefix-32", 32, "77c104d198f1ad17117f5738f5361bf30661b6f281325a6bec415725584b5306")
	testMixedReiReadingOriginal(t, "unit-counter-tail-32", 32, "b0e6b7dca074f19ea11e4ac2c8ea62612f0e873d14ef1a07fe9df5e70ca42bcd")
}
func TestUnitCounterTailReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_UNIT_COUNTER_TAIL_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_UNIT_COUNTER_TAIL_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/unit_counter_tail_wave_original.json", 14, 4)
}
