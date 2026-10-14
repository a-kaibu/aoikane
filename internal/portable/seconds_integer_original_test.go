package portable

import (
	"os"
	"testing"
)

func TestSecondsIntegerReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "seconds-integer-20", 20, "99faec8a2c959d056d94e9733e89d25fc9b7bfba28ef6c293d7809d56b4418c4")
}

func TestSecondsIntegerBoundaryReadingOriginal(t *testing.T) {
	if os.Getenv("AOIKANE_SECONDS_INTEGER_BOUNDARY_READING_AUDIT") != "1" {
		t.Skip("set AOIKANE_SECONDS_INTEGER_BOUNDARY_READING_AUDIT=1 for the complete boundary corpus")
	}
	testMixedReiReadingOriginal(t, "seconds-integer-boundary-16", 16, "7a25999e28339bfbb2967bdebbec4ab6351ae8fb812ce1934460bd1f0e72b84e")
}

func TestSecondsIntegerReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_SECONDS_INTEGER_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_SECONDS_INTEGER_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/seconds_integer_wave_original.json", 8, 4)
}
