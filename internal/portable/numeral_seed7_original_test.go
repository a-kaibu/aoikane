package portable

import (
	"os"
	"testing"
)

// Retain all original inputs, including every unresolved difference.
func TestRandomNumeralSeed7ReadingOriginal(t *testing.T) {
	if os.Getenv("AOIKANE_NUMERAL_SEED7_READING_AUDIT") != "1" {
		t.Skip("set AOIKANE_NUMERAL_SEED7_READING_AUDIT=1 to audit all inputs including unresolved readings")
	}
	testMixedReiReadingOriginal(t, "random-numeral-seed7", 128, "1a2f5d31eef9d486ba0a12fa63328ab7f99f0a02ed2c2599931ab409f1d60351")
}
