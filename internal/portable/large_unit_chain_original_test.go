package portable

import (
	"os"
	"testing"
)

func TestLargeUnitChainReadingOriginal(t *testing.T) {
	t.Run("chain12", func(t *testing.T) {
		testMixedReiReadingOriginal(t, "large-unit-chain-12", 12, "c64d71186d91589ce4d7b4489f6cc3741358be995e83d9b7bffb79a372466779")
	})
	t.Run("boundary20", func(t *testing.T) {
		testMixedReiReadingOriginal(t, "large-unit-chain-boundary-20", 20, "6c7e9c0d352f6a69d7d0ebd24b55bd7f455534ca7603bbf7a0fcf7813e7eba52")
	})
}

func TestLargeUnitChainReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_LARGE_UNIT_CHAIN_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_LARGE_UNIT_CHAIN_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/large_unit_chain_wave_original.json", 6)
}
