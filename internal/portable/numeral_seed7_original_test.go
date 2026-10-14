package portable

import "testing"

// Retain all 128 original inputs in the regular regression without filtering.
func TestRandomNumeralSeed7ReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "random-numeral-seed7", 128, "1a2f5d31eef9d486ba0a12fa63328ab7f99f0a02ed2c2599931ab409f1d60351")
}
