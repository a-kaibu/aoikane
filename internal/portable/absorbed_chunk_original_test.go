package portable

import (
	"os"
	"testing"
)

func TestAbsorbedChunkReadingOriginal(t *testing.T) {
	testMixedReiReadingOriginal(t, "absorbed-chunk-32", 32, "60975ff6cfce5725ccb18febd289ce41815232d46395c766cc0b73592ea93957")
}

func TestAbsorbedChunkReferenceWAV(t *testing.T) {
	if os.Getenv("AOIKANE_ABSORBED_CHUNK_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_ABSORBED_CHUNK_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	testMixedReiReferenceWAV(t, "testdata/absorbed_chunk_wave_original.json", 6)
}
