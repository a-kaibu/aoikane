package portable

import (
	"os"
	"testing"
)

func TestReiSecondsReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "rei-seconds-month-25", 25, "d97a38d1be939775f20bee960e52d3afefc520884f2fd7f91461e27a28244164")
	testMixedReiReadingOriginal(t, "random-numeral-seed1", 128, "af6b3432ac36eea05c0afe2f737a9ee0a426978ad719bccc3117d9c490416f7c")
	testMixedReiReadingOriginal(t, "numeric-continuation-24", 24, "68e3387a7ea15d58fe3171dc61776eb5d2d0b92baa31d6e4d33c556458d39c77")
}

func TestReiSecondsReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_REI_SECONDS_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_REI_SECONDS_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/rei_seconds_wave_original.json", 8)
}
