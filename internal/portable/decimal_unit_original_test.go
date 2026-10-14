package portable

import (
	"os"
	"testing"
)

func TestDecimalUnitReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "decimal-unit-context-48", 48, "66b3d7267ed30eda36f09bb41e14e6a71ad6c221e76b624291710a1738c49934")
}

func TestDecimalUnitReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_DECIMAL_UNIT_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_DECIMAL_UNIT_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/decimal_unit_wave_original.json", 8)
}
