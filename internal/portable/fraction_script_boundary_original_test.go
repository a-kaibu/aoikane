package portable

import (
	"os"
	"testing"
)

func TestFractionScriptBoundaryReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "fraction-script-boundary-33", 33, "3b4787ae7d1ca66303d18a90b88d79d3706143c57b96b97b3c5f34cf7b153d3a")
}
func TestFractionScriptBoundaryReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_FRACTION_SCRIPT_BOUNDARY_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_FRACTION_SCRIPT_BOUNDARY_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/fraction_script_boundary_wave_original.json", 6, 4)
}
