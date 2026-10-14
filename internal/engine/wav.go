package engine

import (
	"encoding/binary"
	"fmt"
)

// WAV wraps the engine's mono signed 16-bit little-endian PCM.
func WAV(pcm []byte, sampleRate int) ([]byte, error) {
	if len(pcm) == 0 || len(pcm)%2 != 0 || uint64(len(pcm)) > 0xffffffff-36 || sampleRate < 8000 || sampleRate > 192000 {
		return nil, fmt.Errorf("invalid PCM output (%d bytes, %d Hz)", len(pcm), sampleRate)
	}
	b := make([]byte, 44+len(pcm))
	copy(b, "RIFF")
	copy(b[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(b[4:], uint32(36+len(pcm)))
	binary.LittleEndian.PutUint32(b[16:], 16)
	binary.LittleEndian.PutUint16(b[20:], 1)
	binary.LittleEndian.PutUint16(b[22:], 1)
	binary.LittleEndian.PutUint32(b[24:], uint32(sampleRate))
	binary.LittleEndian.PutUint32(b[28:], uint32(sampleRate*2))
	binary.LittleEndian.PutUint16(b[32:], 2)
	binary.LittleEndian.PutUint16(b[34:], 16)
	copy(b[36:], "data")
	binary.LittleEndian.PutUint32(b[40:], uint32(len(pcm)))
	copy(b[44:], pcm)
	return b, nil
}
