package portable

import (
	"os"
	"testing"
)

func TestConnectorSecondsReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "connector-seconds-20", 20, "40f3a1814b851e79655fe16bfa6ca024c5a6dd5a83895f26df5ffde91b03434f")
	testMixedReiReadingOriginal(t, "connector-seconds-boundary-12", 12, "d35a85164dbdfe421cc76f0eb315fc31e4aaf79655764449ebd42bd9418c7e17")
	testMixedReiReadingOriginal(t, "connector-seconds-counter-8", 8, "8433af96c4b122ad4680a3d2585fa3956b0f126b96f0da148c41c4dd6b55f78d")
}

func TestConnectorSecondsReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_CONNECTOR_SECONDS_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_CONNECTOR_SECONDS_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/connector_seconds_wave_original.json", 8)
}
