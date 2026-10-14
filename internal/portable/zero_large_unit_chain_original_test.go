package portable

import (
	"os"
	"testing"
)

func TestZeroLargeUnitChainReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "zero-large-unit-chain-20", 20, "c4b60aaf46b619a079fe2a4c562b38f7f6380b304b8675be9c52fcb9bf5260ac")
	testMixedReiReadingOriginal(t, "zero-large-unit-chain-boundary-8", 8, "8d5a9fc52d55936b3b72c160adadee35bc51752e8a5d840bd5f5f29d955ba9e9")
}

func TestZeroLargeUnitChainReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_ZERO_LARGE_UNIT_CHAIN_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_ZERO_LARGE_UNIT_CHAIN_REFERENCE_AUDIT=1 for all original WAV references")
	}
	testMixedReiReferenceWAV(t, "testdata/zero_large_unit_chain_wave_original.json", 8)
}
