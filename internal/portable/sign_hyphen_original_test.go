package portable

import (
	"os"
	"testing"
)

func TestSignHyphenReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "sign-hyphen-stream-32", 32, "d06c6265e0104904809d461e2da3b65c82a27e87b8d33ea65584bb2c59f2ebc4")
	testMixedReiReadingOriginal(t, "sign-hyphen-boundary-16", 16, "1e648853c1734e96c324957742a4e6e6e27a1d313a170bfe639103302d0de6d1")
	testMixedReiReadingOriginal(t, "sign-hyphen-particle-controls-16", 16, "b6d5456ac9737e20569586c384c1a29513cea120d571dc5ab2322e8eddb5adb3")
	testMixedReiReadingOriginal(t, "seconds-prefix-accent-33", 33, "34e978f614c7cb614512c55e5a154e8b4aacb65a743f0fd45495f825ee1c5d28")
	testMixedReiReadingOriginal(t, "random-numeral-seed12-long", 128, "7276a52b63f3fd0267e834f19cd91a6b053c58b9af4c721d70d5b9308d703c5b")
}

func TestSignHyphenReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_SIGN_HYPHEN_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_SIGN_HYPHEN_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/sign_hyphen_wave_original.json", 12, 8)
}
