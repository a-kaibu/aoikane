package portable

import (
	"os"
	"testing"
)

func TestCounterWordReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "counter-word-mode-33", 33, "dc228bb4e9c744d52bb7737c880719e5c7ef9af021127b13993441554512f9fb")
}
func TestCounterWordReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_COUNTER_WORD_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_COUNTER_WORD_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/counter_word_wave_original.json", 8, 6)
}
