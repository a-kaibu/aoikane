package audio

import (
	"bytes"
	"fmt"

	"github.com/a-kaibu/aoikane/third_party/shine-mp3/pkg/mp3"
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
	validRate := false
	for _, rate := range []int{32000, 44100, 48000} {
		if buffer.Format.SampleRate == rate {
			validRate = true
		}
	}
	if !validRate {
		return nil, fmt.Errorf("MP3 input must use 32000, 44100 or 48000 Hz (got %d)", buffer.Format.SampleRate)
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
