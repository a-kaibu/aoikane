package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/gomlx/gomlx/core/tensors"
)

type vocoderFixture struct {
	Frames  int
	Inputs  map[string][]float32
	Wave    []float32
	Seconds float64
	path    string
}

func loadVocoderFixture(path string) (*vocoderFixture, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f := &vocoderFixture{path: path}
	if err := json.Unmarshal(data, f); err != nil {
		return nil, err
	}
	if f.Frames < 1 || f.Frames > 110 || len(f.Wave) != f.Frames*240 {
		return nil, fmt.Errorf("invalid vocoder fixture frame count")
	}
	return f, nil
}

func (f *vocoderFixture) save(outputs []*tensors.Tensor) error {
	if len(outputs) != 1 || outputs[0].Shape().Size() != len(f.Wave) {
		return fmt.Errorf("unexpected vocoder waveform shape")
	}
	var pcm []float32
	err := tensors.ConstFlatData(outputs[0], func(data []float32) { pcm = append([]float32(nil), data...) })
	if err != nil {
		return err
	}
	base := strings.TrimSuffix(f.path, ".json")
	if err := writeWAV(base+"-current.wav", f.Wave); err != nil {
		return err
	}
	return writeWAV(base+"-gomlx.wav", pcm)
}

func writeWAV(path string, samples []float32) error {
	const hz = 48000
	buf := make([]byte, 0, 44+2*len(samples))
	buf = append(buf, "RIFF"...)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(36+2*len(samples)))
	buf = append(buf, "WAVEfmt "...)
	buf = binary.LittleEndian.AppendUint32(buf, 16)
	buf = binary.LittleEndian.AppendUint16(buf, 1)
	buf = binary.LittleEndian.AppendUint16(buf, 1)
	buf = binary.LittleEndian.AppendUint32(buf, hz)
	buf = binary.LittleEndian.AppendUint32(buf, 2*hz)
	buf = binary.LittleEndian.AppendUint16(buf, 2)
	buf = binary.LittleEndian.AppendUint16(buf, 16)
	buf = append(buf, "data"...)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(2*len(samples)))
	for _, value := range samples {
		sample := int16(math.Round(math.Max(-1, math.Min(1, float64(value))) * 32767))
		buf = binary.LittleEndian.AppendUint16(buf, uint16(sample))
	}
	return os.WriteFile(path, buf, 0600)
}
