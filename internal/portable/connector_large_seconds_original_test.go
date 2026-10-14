package portable

import (
	"os"
	"testing"
)

func TestConnectorLargeSecondsReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "connector-large-seconds-20", 20, "482d57a85153680782608955204ade4de2c7246853cb5a6a078a989cd097c71d")
}
func TestConnectorLargeSecondsBoundaryReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "connector-large-seconds-boundary-12", 12, "2b23201be22528c02cb654569063ddb5edab336e04932a3a362339c89285384b")
}
func TestConnectorLargeSecondsReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_CONNECTOR_LARGE_SECONDS_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_CONNECTOR_LARGE_SECONDS_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/connector_large_seconds_wave_original.json", 8, 4)
}
