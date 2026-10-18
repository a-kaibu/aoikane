package portable

import (
	"os"
	"testing"
)

func TestSecondsUnitTailReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "seconds-zero-tail-33", 33, "d4b8e7f1d0c998c09d48c1f3af372f2dcc8c785b84cd89eb55b8a556b324b88d")
	testMixedReiReadingOriginal(t, "seconds-unit-tail-33", 33, "78caae36cb54c7c4549dbe6b3cf378ad457bafb4fd33b0945028f0332280f4a7")
	testMixedReiReadingOriginal(t, "random-numeral-seed11-long", 128, "771e12167a609758ba6101dcea3432b5abfe5d9177fe50a49066e386c2d4ad77")
}

func TestSecondsUnitTailReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_SECONDS_UNIT_TAIL_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_SECONDS_UNIT_TAIL_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/seconds_unit_tail_wave_original.json", 12, 6)
}
