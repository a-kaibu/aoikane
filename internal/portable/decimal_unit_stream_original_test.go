package portable

import (
	"os"
	"testing"
)

func TestDecimalUnitStreamReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "decimal-unit-stream-16", 16, "cd87e0c5939d1d5ba05e6785be535d02d96858278779838e0374461b944ab089")
}

func TestDecimalUnitStreamReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_DECIMAL_UNIT_STREAM_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_DECIMAL_UNIT_STREAM_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/decimal_unit_stream_wave_original.json", 6)
}
