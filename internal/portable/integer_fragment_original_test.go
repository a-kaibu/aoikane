package portable

import (
	"os"
	"testing"
)

func TestIntegerFragmentReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "integer-fragment-68", 68, "accef6e01e2aede83efbf12b8bad12e6d487ae9eae51ac7346245d2519423db0")
}

func TestIntegerFragmentReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_INTEGER_FRAGMENT_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_INTEGER_FRAGMENT_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/integer_fragment_wave_original.json", 6)
}

func TestIntegerFragmentLongReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_INTEGER_FRAGMENT_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_INTEGER_FRAGMENT_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/integer_fragment_long_wave_original.json", 4)
}

func TestIntegerFragmentLongReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "integer-fragment-long-16", 16, "f5ce0d2741c11a2c6a1050715efc8a1d466134b974af83b108cc588a3a35053e")
}
