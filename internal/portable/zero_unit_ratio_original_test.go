package portable

import (
	"os"
	"testing"
)

func TestZeroUnitRatioReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "zero-unit-ratio-16", 16, "a9ebd50c30b04a302735906f3fb2325d3d8a10929c5621158423d8390f6a937d")
	testMixedReiReadingOriginal(t, "zero-unit-ratio-boundary-32", 32, "bb5260ccd02b82aa75ac91a10b753af32eebac0823f4ee8e25e61d7721a04e5c")
	testMixedReiReadingOriginal(t, "zero-unit-ratio-word-24", 24, "c11cb6329bdc210d97962d336700e433b00ab1f98bf31cb8556e9c793f1bd3e5")
}

func TestZeroUnitRatioReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_ZERO_UNIT_RATIO_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_ZERO_UNIT_RATIO_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/zero_unit_ratio_wave_original.json", 12, 8)
}
