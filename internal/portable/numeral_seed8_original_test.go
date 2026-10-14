package portable

import "testing"

// Compare every input of the longer numeral corpus in the normal regression.
func TestRandomNumeralSeed8ReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "random-numeral-seed8", 128, "369f67dc469eea9a0dacd5ba0a52901e15917b464b91ba03cc7fe3728ce37b18")
}
