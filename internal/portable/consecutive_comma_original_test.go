package portable

import (
	"os"
	"testing"
)

func TestConsecutiveCommaReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "consecutive-comma-16", 16, "db5564460500124533ffb6074727504fc3d7fc6b14ce2eea94722c61fe5ac63e")
}

func TestConsecutiveCommaCounterReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "consecutive-comma-counter-16", 16, "a3c2545af4e3e0b310bb70bfbe737404af332b81775bf1e2e7ec9f2d124c4a76")
}

func TestConsecutiveCommaReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_CONSECUTIVE_COMMA_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_CONSECUTIVE_COMMA_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/consecutive_comma_wave_original.json", 8, 4)
}
