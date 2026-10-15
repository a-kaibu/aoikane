package portable

import (
	"os"
	"testing"
)

func TestDecimalRangeConnectorReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "decimal-range-connector-24", 24, "f42fcde8740d688cc0886a9510215e6cb570b193d3ef266c326dc1f87d001be0")
}

func TestDecimalRangeUnitReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "decimal-range-unit-24", 24, "c29022550b7af5182246b6fb5332306084ce3cfada9e135740b9dd5b24e5ba8c")
}

func TestDecimalRangeConnectorReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_DECIMAL_RANGE_CONNECTOR_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_DECIMAL_RANGE_CONNECTOR_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/decimal_range_connector_wave_original.json", 14, 6)
}
