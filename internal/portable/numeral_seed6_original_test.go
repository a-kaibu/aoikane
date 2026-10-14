package portable

import "testing"

// Keep the entire original corpus, including every unresolved input.
func TestRandomNumeralSeed6ReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "random-numeral-seed6", 128, "4ed3faf9bd9156ae1ddfb2e7843c9855ed095de1a0247ff982c687856f5673b9")
}
