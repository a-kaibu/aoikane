package portable

import (
	"os"
	"testing"
)

func TestZeroSecondsReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "zero-seconds-18", 18, "489c7aa1d4cf83108d907b09f641c016dcb3e2c99dcbea524d0f7f148ad5d7ad")
	testMixedReiReadingOriginal(t, "zero-seconds-standalone-8", 8, "a0632e66afea0b0cbbc3caa86b430037c29a25c0592e29933a859f3496116e78")
}

func TestZeroSecondsReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_ZERO_SECONDS_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_ZERO_SECONDS_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/zero_seconds_wave_original.json", 8)
}
