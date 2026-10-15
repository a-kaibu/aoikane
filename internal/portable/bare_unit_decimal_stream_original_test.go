package portable

import (
	"os"
	"testing"
)

func TestBareUnitDecimalStreamReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "bare-unit-decimal-24", 24, "012b55f84583d97a16d3079095be55cae0ba0e3894102f331729e2b832c98dbd")
}

func TestBareUnitDecimalBoundaryReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "bare-unit-decimal-boundary-16", 16, "aea3d79606b21a75281f6aa60969da67840684cf9210e5508de0b0bcae9ae4a2")
}

func TestRandomNumeralSeed9ReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "random-numeral-seed9", 128, "1fb50a3e4f8f5b227262546bd6783f2621f9925b165100e3ea9347e71eb0cf40")
}

func TestRandomNumeralSeed9LongReadingOriginal(t *testing.T) {
	if os.Getenv("AOIKANE_NUMERAL_SEED9_LONG_READING_AUDIT") != "1" {
		t.Skip("set AOIKANE_NUMERAL_SEED9_LONG_READING_AUDIT=1 for the complete unresolved corpus")
	}
	testMixedReiReadingOriginal(t, "random-numeral-seed9-long", 128, "ea03c7da8f0af15d570e63c97965e1d59efb68967877cd4c30ec3986bb8d19bd")
}

func TestBareUnitDecimalStreamReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_BARE_UNIT_DECIMAL_STREAM_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_BARE_UNIT_DECIMAL_STREAM_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/bare_unit_decimal_stream_wave_original.json", 10, 8)
}
