package portable

import (
	"os"
	"testing"
)

func TestReiDaysReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "rei-days-16", 16, "f1b403dac4efe0640d6406c6408cdbabebcd5050b9f7085fb954ec278131bea7")
}

func TestReiDaysBoundaryReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "rei-days-boundary-8", 8, "b0f37a50e2747f931f6c511c65a01671840f28364fc703bcdd8955cda1526059")
}

func TestReiDaysReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_REI_DAYS_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_REI_DAYS_REFERENCE_AUDIT=1 for all original WAV references")
	}
	testMixedReiReferenceWAV(t, "testdata/rei_days_wave_original.json", 8)
}
