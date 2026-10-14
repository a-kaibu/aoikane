package portable

import (
	"os"
	"testing"
)

func TestRangeFractionCounterReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "unit-fraction-counter-8", 8, "dbc1a539ad959ef277a9b3d865d3ca791700b69d8e05b51a9f4701f40d58ab48")
	testMixedReiReadingOriginal(t, "range-fraction-counter-16", 16, "0accde870738d5aeb304e23876774500718fb74ffe63dfd411ff4b98c129a721")
	testMixedReiReadingOriginal(t, "ratio-decimal-counter-8", 8, "82686f523d5ef0f6421d92510ac1f94185ef94c3561645bdb5e88f527ed874df")
	testMixedReiReadingOriginal(t, "colon-literal-counter-8", 8, "d9ec140465d83773639e8aa7d059860009e21c05256b0d82b05ce6dab9ae7074")
	testMixedReiReadingOriginal(t, "range-fraction-plain-control-1", 1, "b982aed70014fdbb8f77e2d059cf911cc794746dcebeff059b567d91c73bf546")
	testMixedReiReadingOriginal(t, "range-fraction-odd-8", 8, "173a309c28a5487d6db744d7a4c941fefed760de3447f72cedc2cbfde64af7fd")
}

func TestRangeFractionCounterReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_RANGE_FRACTION_COUNTER_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_RANGE_FRACTION_COUNTER_REFERENCE_AUDIT=1 for all original WAV references")
	}
	testMixedReiReferenceWAV(t, "testdata/range_fraction_counter_wave_original.json", 16)
}
