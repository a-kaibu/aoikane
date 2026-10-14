package portable

import (
	"os"
	"testing"
)

func TestConnectorLargeUnitReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "connector-large-unit-12", 12, "96c6fc44106330b03628d353434f2c371098cb0007b7e845ef60dc2665424c40")
}

func TestConnectorLargeUnitReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_CONNECTOR_LARGE_UNIT_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_CONNECTOR_LARGE_UNIT_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/connector_large_unit_wave_original.json", 6)
}
