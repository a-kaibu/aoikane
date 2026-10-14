package portable

import (
	"os"
	"testing"
)

func TestGCounterReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "g-counter-24", 24, "976e9528dd8b0599e07cc54e6276e0748e3767be1a54a8bbe288f76bbe1c30ec")
	testMixedReiReadingOriginal(t, "g-counter-boundary-8", 8, "0cc6f2a976ea031e156a5202e5f55413777ae83975e9e5adc5d4eedd717f8c63")
}

func TestGCounterReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_G_COUNTER_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_G_COUNTER_REFERENCE_AUDIT=1 for all original WAV references")
	}
	testMixedReiReferenceWAV(t, "testdata/g_counter_wave_original.json", 12)
}
