package portable

import (
	"os"
	"testing"
)

func TestZeroPointUnitAddressReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "zero-point-unit-address-20", 20, "3a59a7f844d8e25d7a6bf0d55a62d52d3d6b8b8a4272994361ffaf7671f27c07")
}

func TestZeroPointUnitAddressBoundaryReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "zero-point-unit-address-boundary-8", 8, "1f900cd2dabae05c3498b16074bc7610983e28b00986af7aec50af3f16d142bd")
}

func TestZeroPointUnitAddressReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_ZERO_POINT_UNIT_ADDRESS_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_ZERO_POINT_UNIT_ADDRESS_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/zero_point_unit_address_wave_original.json", 8)
}
