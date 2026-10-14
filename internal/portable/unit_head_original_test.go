package portable

import (
	"os"
	"testing"
)

func TestUnitHeadReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "unit-head-56", 56, "82af513646dd3da9abb0b71e5e229377247a832a9e4d37abb7c3e5f99d0e95a8")
}

func TestUnitHeadReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_UNIT_HEAD_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_UNIT_HEAD_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/unit_head_wave_original.json", 6)
}
