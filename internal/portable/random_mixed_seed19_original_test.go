package portable

import "testing"

func TestRandomMixedSeed19ReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "random-mixed-seed19-long", 128, "3085a676836179bfa0331199a3c9fd0329db814c340242204e803d3aa0f385db")
}
