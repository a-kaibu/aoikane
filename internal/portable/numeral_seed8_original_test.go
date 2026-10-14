package portable

import (
	"os"
	"testing"
)

// Preserve the whole longer numeral corpus, including unresolved differences.
func TestRandomNumeralSeed8ReadingOriginal(t *testing.T) {
	if os.Getenv("AOIKANE_NUMERAL_SEED8_READING_AUDIT") != "1" {
		t.Skip("set AOIKANE_NUMERAL_SEED8_READING_AUDIT=1 to compare all original inputs")
	}
	testMixedReiReadingOriginal(t, "random-numeral-seed8", 128, "369f67dc469eea9a0dacd5ba0a52901e15917b464b91ba03cc7fe3728ce37b18")
}
