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

// Keep every invocation from the sentence that exposed powf rounding in the
// regular DLL-free regression, including the thirteen previously equal calls.
func TestCompressorSeed19PowOriginal(t *testing.T) {
	meta, err := os.ReadFile("testdata/compressor_seed19_pow_original.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		Threads, Values int
		Engine          string `json:"engine_sha256"`
		Config          string `json:"config_sha256"`
		SHA             string `json:"reference_sha256"`
		Observed        string `json:"observed_wav_sha256"`
		Ordinary        string `json:"ordinary_wav_sha256"`
		Calls           []struct{ Run, Frames, Samples int }
	}
	if err = json.Unmarshal(meta, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.Threads != 6 || ref.Values != 334800 || len(ref.Calls) != 14 || ref.Engine != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || ref.Observed != "28769406e32b762f656dc0e32999637fe061ae7e9cf7cb3e63eca6a9c8819fe5" || ref.Ordinary != ref.Observed {
		t.Fatal("ordinary original reference differs")
	}
	config, err := os.ReadFile("../../assets/synth/compressor.json")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(config)) != ref.Config {
		t.Fatal("compressor constants differ")
	}
	var c Compressor
	if err = json.Unmarshal(config, &c); err != nil {
		t.Fatal(err)
	}
	compressed, err := os.ReadFile("testdata/compressor_seed19_pow_original.bin.gz")
	if err != nil {
		t.Fatal(err)
	}
	z, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(z)
	if err != nil {
		t.Fatal(err)
	}
	if err = z.Close(); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != ref.SHA {
		t.Fatal("original tensors differ")
	}
	offset, total := 0, 0
	read := func(n int) []float32 {
		if n < 1 || n > (len(raw)-offset)/4 {
			t.Fatal("original tensor is truncated")
		}
		v := make([]float32, n)
		for i := range v {
			v[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[offset+i*4:]))
		}
		offset += n * 4
		return v
	}
	for _, call := range ref.Calls {
		if call.Samples != call.Frames*240 {
			t.Fatal("original frame extent differs")
		}
		wave, power, want := read(call.Samples), read(call.Frames), read(call.Samples)
		got, err := c.Process(wave, power)
		if err != nil {
			t.Fatal(err)
		}
		for i, v := range got {
			if math.Float32bits(v) != math.Float32bits(want[i]) {
				t.Fatalf("run=%d sample=%d got=%08x original=%08x", call.Run, i, math.Float32bits(v), math.Float32bits(want[i]))
			}
		}
		total += len(got)
	}
	if offset != len(raw) || total != ref.Values {
		t.Fatal("not all native invocations were compared")
	}
	t.Logf("all 14 original calls, %d float32 values, bit-identical", total)
}
