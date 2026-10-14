package portable

import (
	"os"
	"testing"
)

func TestGenericZeroStreamReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "generic-zero-stream-17", 17, "72ad586ba3d143d4d7d58a6f22405463ca82aeb4310448e0832fce6523b721cf")
	testMixedReiReadingOriginal(t, "generic-zero-prefix-6", 6, "76431774b980574ae9806f15d1bef7c04e21b9b5c401aac5b8de651854465093")
}

func TestGenericZeroStreamReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_GENERIC_ZERO_STREAM_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_GENERIC_ZERO_STREAM_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/generic_zero_stream_wave_original.json", 6)
}
