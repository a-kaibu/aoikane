package portable

import (
	"os"
	"testing"
)

func TestSecondsBareUnitReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "seconds-bare-unit-32", 32, "7f8f7c9cf2a9c4190f62ca8c2cab20579ec56d7d3967897d6f46c64cb2455b15")
}
func TestSecondsBareUnitReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_SECONDS_BARE_UNIT_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_SECONDS_BARE_UNIT_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/seconds_bare_unit_wave_original.json", 10)
}
