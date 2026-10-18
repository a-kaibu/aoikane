package portable

import (
	"os"
	"testing"
)

func TestLeadingLiteralZeroDecimalReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "leading-zero-decimal-32", 32, "27bff9810b530e4b6b81ac37df561944a8689217d6b32bfe35f56cf8785a2bc6")
	testMixedReiReadingOriginal(t, "literal-zero-decimal-16", 16, "ab65eae127227c3223905fc5b08a4ce03e3db5eafc473a925b74e9f6b603089d")
}
func TestLeadingLiteralZeroDecimalReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_LEADING_LITERAL_ZERO_DECIMAL_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_LEADING_LITERAL_ZERO_DECIMAL_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/leading_literal_zero_decimal_wave_original.json", 12, 6)
}
