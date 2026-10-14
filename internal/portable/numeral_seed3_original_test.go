package portable

import (
	"testing"
)

// Retain all 128 pinned inputs in the normal original-reading regression.
func TestRandomNumeralSeed3ReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "random-numeral-seed3", 128, "d14141f400ce8f6fce4e285f95ba3a080c1fa6d2c552043f8f94b9869c1fa6e8")
}
