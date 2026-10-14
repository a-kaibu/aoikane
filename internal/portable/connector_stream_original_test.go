package portable

import (
	"os"
	"testing"
)

func TestConnectorStreamReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "connector-stream-49", 49, "3ba1b65eebd3ba26c35f59f9d95513df51cbbbac76d18933bed451b165abde07")
	testMixedReiReadingOriginal(t, "point-boundary-20", 20, "25567a4ef30febb26a195b97b806ed8e3e65b300e7e6f39f0026c71d27d2c5b0")
}

func TestConnectorStreamReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_CONNECTOR_STREAM_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_CONNECTOR_STREAM_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/connector_stream_wave_original.json", 14)
}
