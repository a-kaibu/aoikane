package portable

import (
	"os"
	"testing"
)

func TestMixedReiBigUnitReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "mixed-rei-big-unit-32", 32, "0b2115e3a61bf2f279fb0b01b9cc679be48e3e61bfee043b9f2452ff980b311a")
}

func TestMixedReiBigUnitExtendedReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "mixed-rei-big-unit-64", 64, "7d49ed44c37364b75ba0deaa3ae5efedef0d9aad863c28a65c4a08e0e5b0a4ba")
}

func TestMixedReiBigUnitSokuonReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "mixed-rei-big-unit-sokuon-32", 32, "b9519f3c175b2fe5f1d9844362b48a1160554ac29fc5e5c718824d0df87fca46")
}

func TestMixedReiZeroBigUnitReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "mixed-rei-zero-big-unit-32", 32, "4a52cc0893974a03044e8a8f6c75c3d11f488bfd10b86f525c34ce680168f09a")
}

func TestMixedReiBigUnitExtendedReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_MIXED_REI_BIG_UNIT_EXTENDED_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_MIXED_REI_BIG_UNIT_EXTENDED_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/mixed_rei_big_unit_extended_wave_original.json", 8)
}

func TestMixedReiBigUnitSokuonReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_MIXED_REI_BIG_UNIT_SOKUON_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_MIXED_REI_BIG_UNIT_SOKUON_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/mixed_rei_big_unit_sokuon_wave_original.json", 6)
}

func TestMixedReiZeroBigUnitReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_MIXED_REI_ZERO_BIG_UNIT_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_MIXED_REI_ZERO_BIG_UNIT_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/mixed_rei_zero_big_unit_wave_original.json", 8)
}

func TestMixedReiBigUnitReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_MIXED_REI_BIG_UNIT_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_MIXED_REI_BIG_UNIT_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/mixed_rei_big_unit_wave_original.json", 6)
}
