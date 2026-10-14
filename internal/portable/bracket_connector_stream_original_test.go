package portable

import (
	"os"
	"testing"
)

func TestBracketConnectorStreamReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "bracket-connector-stream-20", 20, "e8c9d744dca6f478c9cc409af1a034f4eab7dfab96ba8e9338b7b45c756f9819")
	testMixedReiReadingOriginal(t, "bracket-connector-boundary-8", 8, "91cddaf8ab926c71e7bc912df5de71cd46b620dba617be6f5ad1c2d904f44b78")
	testMixedReiReadingOriginal(t, "bracket-connector-phone-colon-8", 8, "9c771b3715c8ddffe4de44344d9eca8a104f03491038daa86195421ec42b9a60")
}

func TestBracketConnectorStreamReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_BRACKET_CONNECTOR_STREAM_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_BRACKET_CONNECTOR_STREAM_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/bracket_connector_stream_wave_original.json", 8)
}
