package portable

import (
	"os"
	"testing"
)

func TestInteriorUnitSecondsReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "interior-unit-seconds-16", 16, "ecf52d82c01519cb0f479f0e82f9901acf826cd090ca668ea958430f9e29a469")
}

func TestInteriorUnitSecondsReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_INTERIOR_UNIT_SECONDS_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_INTERIOR_UNIT_SECONDS_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/interior_unit_seconds_wave_original.json", 4)
}
