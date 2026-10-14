package portable

import (
	"os"
	"testing"
)

func TestDecimalCounterKeyReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "decimal-counter-key-33", 33, "7fd86c0db638a7ef3a156d2ec26736b62f9c09771fc59a8b2507d7d907b8cc70")
}

func TestDecimalCounterKeyBoundaryReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "decimal-counter-key-boundary-8", 8, "0ee764e8eb5b7f037dd1e393cc652c4e5229cadce7f07c204d9a1653996d476c")
}

func TestDecimalCounterKeyIntegerPrefixReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "decimal-counter-key-range-2", 2, "c6028f4783c855d456fa688de5b164e3f8864e52a93110c40dad61cafa845f09")
}

func TestDecimalCounterKeyReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_DECIMAL_COUNTER_KEY_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_DECIMAL_COUNTER_KEY_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/decimal_counter_key_wave_original.json", 12)
}
