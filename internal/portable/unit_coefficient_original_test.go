package portable

import (
	"os"
	"testing"
)

func TestUnitCoefficientReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "unit-coefficient-accent-20", 20, "c4f8f14797acba4b9c568144455c75d15f802dc548fb4a763fe62f96ea7cfb8b")
	testMixedReiReadingOriginal(t, "unit-coefficient-boundary-32", 32, "a29170d0d2de7ab3248604bb3631d97aa8c1f4fdff64f63737e26dce8608246e")
}

func TestUnitCoefficientReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_UNIT_COEFFICIENT_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_UNIT_COEFFICIENT_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/unit_coefficient_wave_original.json", 8, 4)
}
