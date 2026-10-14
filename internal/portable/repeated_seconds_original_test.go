package portable

import (
	"os"
	"testing"
)

func TestRepeatedSecondsReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "repeated-seconds-20", 20, "51ec6642beddb869b8f098f93787afcef2f46967bf19d92e510465e342686caf")
}

func TestRepeatedSecondsBoundaryReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "repeated-seconds-boundary-12", 12, "0eb40b5610ac548bb65168bcb17a415a874075b03129f60e2e9605e5da103413")
}

func TestRepeatedSecondsAnchorReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "repeated-seconds-anchor-2", 2, "4213f6714c136ce4b9bc2d221f0a5b68113d9c4678b26f2098608499fa4e1992")
}

func TestRepeatedSecondsReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_REPEATED_SECONDS_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_REPEATED_SECONDS_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/repeated_seconds_wave_original.json", 12, 6)
}
