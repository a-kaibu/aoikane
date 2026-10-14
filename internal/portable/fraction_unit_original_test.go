package portable

import (
	"os"
	"testing"
)

func TestFractionUnitReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "fraction-large-unit-11", 11, "4bc174270603f64b48ec7152d160f06dbd14fb5d5a4df9168be66390d6283d05")
	testMixedReiReadingOriginal(t, "fraction-unit-48", 48, "3c821966e144e645a17de2dc08f30e39d7e0cb2962e3f6f5e2721b96db867a8f")
	testMixedReiReadingOriginal(t, "fraction-unit-60", 60, "c03c2fe2f11a5d6ee4fbbd804c7723c7692f07b0e64600e08c6f58d1e4fd18a0")
}

func TestRandomNumeralSeed2ReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "random-numeral-seed2", 128, "118590e280fae23010074255a72581dfb5f0f1a6b3ad7da288dbfd8fd9083cf2")
}

func TestFractionUnitReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_FRACTION_UNIT_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_FRACTION_UNIT_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/fraction_unit_wave_original.json", 8)
}
