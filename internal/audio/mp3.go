package audio

import (
	"bytes"
	"fmt"

	"github.com/braheezy/shine-mp3/pkg/mp3"
	"github.com/go-audio/wav"
)

// Convert uses Shine entirely in Go; no ffmpeg, subprocess or encoder DLL.
func Convert(data []byte, format string) ([]byte, error) {
	if format == "" || format == "wav" {
		return data, nil
	}
	if format != "mp3" {
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
	decoder := wav.NewDecoder(bytes.NewReader(data))
	buffer, err := decoder.FullPCMBuffer()
	if err != nil {
		return nil, fmt.Errorf("decode WAV: %w", err)
	}
	if decoder.BitDepth != 16 || buffer.Format.NumChannels < 1 || buffer.Format.NumChannels > 2 || len(buffer.Data) == 0 {
		return nil, fmt.Errorf("MP3 input must be nonempty 16-bit mono/stereo PCM WAV")
	}
	if buffer.Format.SampleRate == 24000 {
		channels := buffer.Format.NumChannels
		if len(buffer.Data)%channels != 0 {
			return nil, fmt.Errorf("24 kHz WAV has incomplete PCM frames")
		}
		buffer.Data = doubleSampleRate(buffer.Data, channels)
		buffer.Format.SampleRate = 48000
	}
	validRate := false
	for _, rate := range []int{32000, 44100, 48000} {
		if buffer.Format.SampleRate == rate {
			validRate = true
		}
	}
	if !validRate {
		return nil, fmt.Errorf("MP3 input must use 24000, 32000, 44100 or 48000 Hz (got %d)", buffer.Format.SampleRate)
	}
	encoder := mp3.NewEncoder(buffer.Format.SampleRate, buffer.Format.NumChannels)
	frameSize := int(encoder.Mpeg.GranulesPerFrame) * 576 * buffer.Format.NumChannels
	frame := make([]int16, frameSize)
	var output bytes.Buffer
	// EncodeBufferInterleaved avoids Write's stereo chunk-size bug in v0.2.0.
	// One trailing zero frame drains the analysis filter and MDCT delay.
	for offset := 0; offset < len(buffer.Data)+frameSize; offset += frameSize {
		clear(frame)
		for i := 0; i < frameSize && offset+i < len(buffer.Data); i++ {
			frame[i] = int16(buffer.Data[offset+i])
		}
		encoded, n := encoder.EncodeBufferInterleaved(frame)
		output.Write(encoded[:n])
	}
	if output.Len() == 0 {
		return nil, fmt.Errorf("MP3 encoder produced no frames")
	}
	return output.Bytes(), nil
}

// doubleSampleRate inserts the midpoint of a four-sample interpolator.
// Frames are interleaved, and the edge frames are extended for the filter.
func doubleSampleRate(data []int, channels int) []int {
	frames := len(data) / channels
	out := make([]int, len(data)*2)
	for frame := range frames {
		for channel := range channels {
			a := data[max(frame-1, 0)*channels+channel]
			b := data[frame*channels+channel]
			c := data[min(frame+1, frames-1)*channels+channel]
			d := data[min(frame+2, frames-1)*channels+channel]
			out[2*frame*channels+channel] = b
			out[(2*frame+1)*channels+channel] = min(32767, max(-32768, (-a+9*b+9*c-d)/16))
		}
	}
	return out
}
