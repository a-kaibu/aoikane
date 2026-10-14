package portable

import (
	"os"
	"testing"
)

func TestPointUnitTailReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "point-unit-tail-24", 24, "123b44a21d8e45cb7e8c7f21a6286ddd925868b0fc0c04af8c414509cafd9db6")
	testMixedReiReadingOriginal(t, "point-unit-tail-extra-8", 8, "bdc629e7e0e23222ee993b2b7f990e8e52070c856a4078db81dd39e51d7a2a6d")
}

func TestPointUnitTailReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_POINT_UNIT_TAIL_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_POINT_UNIT_TAIL_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/point_unit_tail_wave_original.json", 8)
}
