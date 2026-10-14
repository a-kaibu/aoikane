package portable

import (
	"os"
	"testing"
)

func TestSeed19ZeroDigitReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "seed19-zero-digit-48", 48, "fb53275df4b05239db2f146689c56cad712a8eb5b6a6b279a59b5a73dc9cb29b")
}

func TestSeed19ZeroDigitReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_SEED19_ZERO_DIGIT_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_SEED19_ZERO_DIGIT_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/seed19_zero_digit_wave_original.json", 6)
}
