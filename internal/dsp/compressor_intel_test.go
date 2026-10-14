package dsp

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"testing"
)

// This capture preserves the ordinary original WAV. Only the compressor's
// actual wave and power are replayed; the original output is a reference.
func TestCompressorIntelAVX512Original(t *testing.T) {
	raw, err := os.ReadFile("testdata/compressor_intel_avx512.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		Synthetic       bool
		Values, Threads int
		DLLHash         string `json:"dll_sha256"`
		ConfigHash      string `json:"config_sha256"`
		OriginalHash    string `json:"ordinary_wav_sha256"`
		ObservedHash    string `json:"observed_wav_sha256"`
		ReferenceHash   string `json:"reference_sha256"`
		Hash            string `json:"output_sha256"`
		Calls           []struct{ Run, Frames, Samples int }
	}
	if err = json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.Synthetic || ref.Threads != 1 || ref.Values != 50160 || len(ref.Calls) != 2 || ref.DLLHash != "00ca5c937940a94d052dbe52ec5fb0909db11ffbfccfd6e65e19b468169ff5e5" || ref.OriginalHash != "6be20e8718dc8a1329426baa824b9dabef10315dddc2099a3ad11ffac6032d7d" || ref.ObservedHash != ref.OriginalHash || ref.Hash != "0914f5accd3aa80e1c74eddfa216a123e3e0aeaa9c35e62d8e19c6a8f8bceba4" {
		t.Fatal("ordinary Intel compressor provenance differs")
	}
	config, err := os.ReadFile("../../assets/synth/compressor.json")
	if err != nil {
		t.Fatal(err)
	}
	if ref.ConfigHash != "cc6409b2162eaa00a3262d2518bb18d43a0187140410684860faacd2981d895e" || fmt.Sprintf("%x", sha256.Sum256(config)) != ref.ConfigHash {
		t.Fatal("original compressor config differs")
	}
	var c Compressor
	if err = json.Unmarshal(config, &c); err != nil {
		t.Fatal(err)
	}
	packed, err := os.ReadFile("testdata/compressor_intel_avx512.bin.gz")
	if err != nil {
		t.Fatal(err)
	}
	r, err := gzip.NewReader(bytes.NewReader(packed))
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(io.LimitReader(r, int64(ref.Values)*9))
	if err != nil {
		t.Fatal(err)
	}
	if err = r.Close(); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != ref.ReferenceHash {
		t.Fatal("original compressor reference hash differs")
	}
	offset := 0
	read := func(count int) []float32 {
		if count < 1 || count > (len(data)-offset)/4 {
			t.Fatal("invalid original compressor storage extent")
		}
		values := make([]float32, count)
		for i := range values {
			values[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[offset:]))
			offset += 4
		}
		return values
	}
	h, total := sha256.New(), 0
	for _, call := range ref.Calls {
		if call.Frames < 1 || call.Frames > 110 || call.Samples != call.Frames*240 {
			t.Fatal("invalid original compressor geometry")
		}
		wave, power, want := read(call.Samples), read(call.Frames), read(call.Samples)
		got, err := c.ProcessWithAVX512Unary(wave, power)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != len(want) {
			t.Fatal("original compressor output length differs")
		}
		for i, v := range got {
			if math.Float32bits(v) != math.Float32bits(want[i]) {
				t.Fatalf("run %d sample %d got=%08x want=%08x", call.Run, i, math.Float32bits(v), math.Float32bits(want[i]))
			}
			h.Write(binary.LittleEndian.AppendUint32(nil, math.Float32bits(v)))
		}
		total += len(got)
	}
	if total != ref.Values || offset != len(data) || fmt.Sprintf("%x", h.Sum(nil)) != ref.Hash {
		t.Fatal("original compressor values/count/hash differs")
	}
	t.Logf("Compared %d ordinary Intel compressor values; SHA256 %x", total, h.Sum(nil))
}
