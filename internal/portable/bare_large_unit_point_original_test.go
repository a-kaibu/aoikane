package portable

import (
	"os"
	"testing"
)

func TestBareLargeUnitPointReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "bare-large-unit-point-20", 20, "27822944c08de8e8675c036f35851726ff6457dec5df5b92e51d9a0fcf11d6b9")
	testMixedReiReadingOriginal(t, "right-decimal-range-8", 8, "75ff269e38e2013e92bd9d6a8327ebffb1b19a94ffe7090a65041d1ee7fc5ce6")
	testMixedReiReadingOriginal(t, "dotted-phone-range-8", 8, "954d9667c19875c4406ee900e5fdf5e2de2c8a05a81389dde56b28b53a9c2251")
}

func TestBareLargeUnitPointReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_BARE_LARGE_UNIT_POINT_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_BARE_LARGE_UNIT_POINT_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/bare_large_unit_point_wave_original.json", 12)
}

func TestDottedPhoneUnitReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "dotted-phone-unit-4", 4, "2458ba2c64cdd72dba888ca9ddb9baff42cceac8e3c11f716005747a7d9926de")
}
