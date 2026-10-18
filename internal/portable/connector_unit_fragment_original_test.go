package portable

import (
	"os"
	"testing"
)

func TestConnectorUnitFragmentReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "connector-unit-fragment-32", 32, "dad6f17f99ff85cf00f17774da8c4031a259ac74208b0e5d0b62ab2655f879ab")
	testMixedReiReadingOriginal(t, "connector-unit-context-40", 40, "b31d00f257152462d6b8872a8f2594014dc5cefec1b65176b2aeedba49f89fa3")
}
func TestConnectorUnitFragmentReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_CONNECTOR_UNIT_FRAGMENT_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_CONNECTOR_UNIT_FRAGMENT_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/connector_unit_fragment_wave_original.json", 12, 6)
}
