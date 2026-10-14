package portable

import (
	"os"
	"testing"
)

func TestConnectorBareThousandReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "connector-bare-thousand-16", 16, "fc13691b0dd6a7f36350a5c1d99f0fc8b06211d91b47d2be720c947aaa21f77a")
}

func TestConnectorBareThousandReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_CONNECTOR_BARE_THOUSAND_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_CONNECTOR_BARE_THOUSAND_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/connector_bare_thousand_wave_original.json", 6)
}
