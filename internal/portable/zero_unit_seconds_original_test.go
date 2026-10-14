package portable

import (
	"os"
	"testing"
)

func TestZeroUnitSecondsReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "zero-unit-seconds-18", 18, "0212d81550f50103695c048ce1394bb9429d942e340c61ef6047a3a9cfdf1cc6")
	testMixedReiReadingOriginal(t, "zero-unit-seconds-25", 25, "96add8c12f436bc445f46fea17b8fdc657c417ada09ddf0062ae0537f05bcc56")
	testMixedReiReadingOriginal(t, "zero-unit-seconds-36", 36, "07eb09f15de47d8d64adfa5924d5081e732fa3d6ee5edcbc014e575168713bd4")
}

func TestZeroUnitSecondsReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_ZERO_UNIT_SECONDS_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_ZERO_UNIT_SECONDS_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/zero_unit_seconds_wave_original.json", 10)
}
