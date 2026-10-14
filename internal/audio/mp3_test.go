package audio

import (
	"bytes"
	"encoding/binary"
	"math"
	"strconv"
	"testing"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
)

// Verify complete MP3 framing against the MPEG header rather than merely
// checking the sync bytes. This catches Shine's MPEG-1 mono size regression.
func TestMP3Frames(t *testing.T) {
	for _, channels := range []int{1, 2} {
		for _, rate := range []int{32000, 44100, 48000} {
			t.Run(strconv.Itoa(channels)+"ch-"+strconv.Itoa(rate), func(t *testing.T) {
				var input bytes.Buffer
				enc := wav.NewEncoder(&memoryFile{Buffer: &input}, rate, 16, channels, 1)
				pcm := make([]int, rate/8*channels)
				for i := range pcm {
					pcm[i] = int(10000 * math.Sin(float64(i/channels)*2*math.Pi*440/float64(rate)))
				}
				if e := enc.Write(&audio.IntBuffer{Data: pcm, Format: &audio.Format{NumChannels: channels, SampleRate: rate}, SourceBitDepth: 16}); e != nil {
					t.Fatal(e)
				}
				if e := enc.Close(); e != nil {
					t.Fatal(e)
				}
				out, e := Convert(input.Bytes(), "mp3")
				if e != nil {
					t.Fatal(e)
				}
				frames := 0
				for off := 0; off < len(out); {
					if off+4 > len(out) {
						t.Fatal("truncated header")
					}
					h := binary.BigEndian.Uint32(out[off:])
					if h>>21 != 0x7ff || h>>17&3 != 1 {
						t.Fatalf("invalid frame at %d", off)
					}
					version := (h >> 19) & 3
					bitIndex := (h >> 12) & 15
					rateIndex := (h >> 10) & 3
					if bitIndex == 0 || bitIndex == 15 || rateIndex == 3 {
						t.Fatal("invalid header indexes")
					}
					bitrate := []int{0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320}[bitIndex]
					sampleRate := []int{44100, 48000, 32000}[rateIndex]
					factor := 144000
					if version != 3 {
						bitrate = []int{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160}[bitIndex]
						factor = 72000
						sampleRate /= 2
						if version == 0 {
							sampleRate /= 2
						}
					}
					if sampleRate != rate {
						t.Fatalf("wrong sample rate %d", sampleRate)
					}
					mono := h>>6&3 == 3
					if mono != (channels == 1) {
						t.Fatal("wrong channel mode")
					}
					size := factor*bitrate/sampleRate + int(h>>9&1)
					if off+size > len(out) {
						t.Fatal("truncated frame")
					}
					off += size
					frames++
				}
				if frames < 2 {
					t.Fatal("missing MP3 frames")
				}
			})
		}
	}
}

// wav.Encoder requires seeking to finalize its RIFF header.
type memoryFile struct {
	*bytes.Buffer
	pos int64
}

func (m *memoryFile) Write(p []byte) (int, error) {
	b := m.Bytes()
	end := m.pos + int64(len(p))
	if end > int64(len(b)) {
		m.Buffer.Write(make([]byte, end-int64(len(b))))
		b = m.Bytes()
	}
	copy(b[m.pos:], p)
	m.pos = end
	return len(p), nil
}
func (m *memoryFile) Seek(off int64, whence int) (int64, error) {
	switch whence {
	case 0:
		m.pos = off
	case 1:
		m.pos += off
	case 2:
		m.pos = int64(m.Len()) + off
	}
	return m.pos, nil
}
