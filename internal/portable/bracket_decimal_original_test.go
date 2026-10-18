package portable

import (
	"os"
	"testing"
)

func TestBracketDecimalReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "bracket-decimal-stream-32", 32, "486500f11c4cf43ad1c06c3bdf494e03defaa6116f017121817c0630123aac5e")
	testMixedReiReadingOriginal(t, "bracket-decimal-phone-16", 16, "d434f877f1c14eac9ce8e2833242c169a2504eb188322cfde9e5fb52a98d8730")
}
func TestBracketDecimalReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_BRACKET_DECIMAL_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_BRACKET_DECIMAL_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/bracket_decimal_wave_original.json", 12, 6)
}
