package portable

import (
	"os"
	"testing"
)

func TestLeadingZeroFragmentsReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "point-leading-zero-12", 12, "7981b5260c592174233f003b357979287332d46fa247aafb2549a537899cbf17")
	testMixedReiReadingOriginal(t, "point-leading-zero-24", 24, "41224d85a06eeadb66924b26799281a56aef0ef36fe70f4d06e279766efd05a0")
	testMixedReiReadingOriginal(t, "unit-digit-boundary-24", 24, "77d009459f38736b51b92d7266f4ae1412adbd4a12eef2f0c83bab8dfdc33b87")
	testMixedReiReadingOriginal(t, "unit-leading-zero-counter-12", 12, "85704f91ea78d7751e51727868dc6f08f18d66835507d7027499de532e080a99")
	testMixedReiReadingOriginal(t, "unit-leading-zero-counter-extra-12", 12, "c964a2ebfaa4fd5feabbfcfab9b0ad02f2010eaf52e7bdea192e8612f816a004")
}

func TestLeadingZeroFragmentsReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_LEADING_ZERO_FRAGMENTS_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_LEADING_ZERO_FRAGMENTS_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/leading_zero_fragments_wave_original.json", 10)
}
