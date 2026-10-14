package portable

import (
	"os"
	"testing"
)

func TestC2CounterReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "c2-counter-16", 16, "29481d2ba40edc8721e7ef17a0fc33cee18bf57c9a3f58163066633b3e64e600")
}

func TestC2CounterReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_C2_COUNTER_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_C2_COUNTER_REFERENCE_AUDIT=1 for all original WAV references")
	}
	testMixedReiReferenceWAV(t, "testdata/c2_counter_wave_original.json", 6)
}
