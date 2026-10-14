package portable

import "testing"

// Retain all pinned inputs, including known failures.
func TestRandomNumeralSeed5ReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "random-numeral-seed5", 128, "b6fe30729a7c34bc40591be9d8f597f8957143ac9dd5fb9adc5d19667191ea26")
}
