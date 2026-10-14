package portable

import (
	"os"
	"testing"
)

func TestUnitReiTailReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "unit-rei-tail-48", 48, "cca571ac231562035947712114ff350369ac38df51a74f5cdeeceee850330094")
}

func TestUnitReiTailReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_UNIT_REI_TAIL_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_UNIT_REI_TAIL_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/unit_rei_tail_wave_original.json", 8)
}
