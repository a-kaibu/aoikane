package portable

import (
	"os"
	"testing"
)

// Preserve the first seed-19 PCM mismatch as a strict DLL-free replay.
// This audit remains failing until synthesis reproduces the original bytes.
func TestSeed19Case117ReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_SEED19_CASE117_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_SEED19_CASE117_REFERENCE_AUDIT=1 for strict original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/seed19_case117_wave_original.json", 2)
}
