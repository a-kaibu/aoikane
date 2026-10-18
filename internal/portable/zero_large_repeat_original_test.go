package portable

import (
	"os"
	"testing"
)

func TestZeroLargeRepeatReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "zero-large-repeat-32", 32, "7a0a39076d2894239e025ae87c8d2f4c2d22513b021b3aa6aaeac91ad19e1483")
	testMixedReiReadingOriginal(t, "zero-large-scope-32", 32, "5209860bd385cd25225c8dbdd716f366db7864ad26ba5c2a377d7750da9bb2ca")
}

func TestZeroLargeRepeatReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_ZERO_LARGE_REPEAT_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_ZERO_LARGE_REPEAT_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/zero_large_repeat_wave_original.json", 12)
}
