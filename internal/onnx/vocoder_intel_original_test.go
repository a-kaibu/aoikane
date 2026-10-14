package onnx

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
	"slices"
	"testing"
)

type intelVocoderCase struct {
	Run, Frames int
	WaveValues  int              `json:"wave_values"`
	WaveHash    string           `json:"wave_sha256"`
	Inputs      map[string]Value `json:"-"`
	Wave        Value            `json:"-"`
}

func loadIntelVocoderReference(t *testing.T) []intelVocoderCase {
	t.Helper()
	raw, err := os.ReadFile("testdata/vocoder_intel_original.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		CPU            string
		Threads        int    `json:"original_threads"`
		ModelHash      string `json:"model_sha256"`
		DLLHash        string `json:"dll_sha256"`
		WaveHash       string `json:"ordinary_wav_sha256"`
		StorageHash    string `json:"reference_sha256"`
		OutputHash     string `json:"native_wave_sha256"`
		ScriptBytes    int    `json:"torchscript_bytes"`
		ScriptVerified bool   `json:"torchscript_sha256_verified"`
		Values         int
		WaveValues     int `json:"wave_values"`
		Cases          []intelVocoderCase
	}
	if err = json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.CPU != "Intel64 Family 6 Model 207 Stepping 2, GenuineIntel" || ref.Threads != 1 || ref.ModelHash != "ff245a6149817c01c2d559e49087892623116766d1f3c95e7996bac23e6887e0" || ref.DLLHash != "00ca5c937940a94d052dbe52ec5fb0909db11ffbfccfd6e65e19b468169ff5e5" || ref.WaveHash != "6be20e8718dc8a1329426baa824b9dabef10315dddc2099a3ad11ffac6032d7d" || ref.ScriptBytes != 9836696 || ref.ScriptVerified || ref.Values != 63954 || ref.WaveValues != 50160 || len(ref.Cases) != 2 {
		t.Fatal("ordinary Intel vocoder reference provenance differs")
	}
	model, err := os.ReadFile("../../assets/synth/vocoder.onnx")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(model)) != ref.ModelHash {
		t.Fatal("Go vocoder ONNX differs")
	}
	compressed, err := os.ReadFile("testdata/vocoder_intel_original.bin.gz")
	if err != nil {
		t.Fatal(err)
	}
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(io.LimitReader(reader, 255817))
	if err != nil {
		t.Fatal(err)
	}
	if err = reader.Close(); err != nil {
		t.Fatal(err)
	}
	if len(data) != 255816 || ref.StorageHash != "e9d441ac9c931dd18bff4c3b994a2ca4a6e491c22c49263e1525ff00d9d7fa79" || fmt.Sprintf("%x", sha256.Sum256(data)) != ref.StorageHash {
		t.Fatal("native storage differs")
	}
	take := func(channels, frames int) Value {
		count := channels * frames
		if count < 1 || count > len(data)/4 {
			t.Fatal("truncated native storage")
		}
		v := Value{Kind: 1, Shape: []int64{1, int64(channels), int64(frames)}, F: make([]float32, count)}
		for i := range v.F {
			v.F[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[4*i:]))
		}
		data = data[count*4:]
		return v
	}
	h := sha256.New()
	var bits [4]byte
	for i := range ref.Cases {
		c := &ref.Cases[i]
		frames := []int{110, 99}[i]
		if c.Run != []int{496, 505}[i] || c.Frames != frames || c.WaveValues != frames*240 {
			t.Fatal("native call geometry differs")
		}
		c.Inputs = map[string]Value{}
		for j, name := range []string{"f0", "bap", "spc"} {
			c.Inputs[name] = take([]int{1, 5, 60}[j], frames)
		}
		c.Wave = take(1, c.WaveValues)
		ch := sha256.New()
		for _, x := range c.Wave.F {
			binary.LittleEndian.PutUint32(bits[:], math.Float32bits(x))
			ch.Write(bits[:])
			h.Write(bits[:])
		}
		if fmt.Sprintf("%x", ch.Sum(nil)) != c.WaveHash {
			t.Fatal("native call output differs")
		}
	}
	if len(data) != 0 || ref.OutputHash != "c5b7764e6ffe882ef9c0a830ef7a3abbad8a228810995f26e49e923199c74681" || fmt.Sprintf("%x", h.Sum(nil)) != ref.OutputHash {
		t.Fatal("native combined output differs")
	}
	return ref.Cases
}

func TestOriginalIntelVocoderReferenceStorage(t *testing.T) {
	loadIntelVocoderReference(t)
}

// Experimental strict audit: current reconstruction still differs. This runs
// from the captured external model inputs, without substituting internal node
// values. Native TorchScript bytes were not retained in this particular trace;
// its SHA256 and per-layer comparison remain unavailable.
func TestOriginalIntelVocoderWholeGraph(t *testing.T) {
	if os.Getenv("AOIKANE_INTEL_VOCODER_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_INTEL_VOCODER_REFERENCE_AUDIT=1 for the unresolved strict Intel vocoder audit")
	}
	cases := loadIntelVocoderReference(t)
	g, err := OpenGraph("../../assets/synth/vocoder.onnx")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		t.Run(fmt.Sprint(c.Run), func(t *testing.T) {
			got, err := g.RunOriginalKernelsWithProfile(c.Inputs, []string{"wave"}, OriginalKernelProfile{Threads: 1, AVX512Convolutions: true, AVX512Unary: true, FMAEmbeddings: true, FMASmallConvolutions: true, FMAProjections: true})
			if err != nil {
				t.Fatal(err)
			}
			v := got["wave"]
			if !slices.Equal(v.Shape, c.Wave.Shape) || len(v.F) != len(c.Wave.F) {
				t.Fatal("native wave geometry differs")
			}
			different, first := 0, -1
			for i, x := range v.F {
				if math.Float32bits(x) != math.Float32bits(c.Wave.F[i]) {
					different++
					if first < 0 {
						first = i
					}
				}
			}
			if different != 0 {
				t.Errorf("ordinary Intel vocoder differs: %d/%d FLOAT values, first=%d", different, len(v.F), first)
			}
		})
	}
}
