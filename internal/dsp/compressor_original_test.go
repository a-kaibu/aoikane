package dsp

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

type originalTensor struct {
	ID, Dtype     int
	Shape, Stride []int64
}

func readOriginalTensor(t *testing.T, root string, ref originalTensor) []float32 {
	t.Helper()
	if ref.Dtype != 6 || len(ref.Shape) != len(ref.Stride) {
		t.Fatal("invalid original FLOAT tensor")
	}
	count, last := int64(1), int64(0)
	for i, d := range ref.Shape {
		if d < 1 || d > (1<<30)/count || ref.Stride[i] < 0 || ref.Stride[i] > (1<<30)/d {
			t.Fatal("invalid original tensor extent")
		}
		count *= d
		last += (d - 1) * ref.Stride[i]
	}
	raw, err := os.ReadFile(filepath.Join(root, fmt.Sprintf("model-%d.bin", ref.ID)))
	if err != nil {
		t.Fatal(err)
	}
	if last >= int64(len(raw))/4 {
		t.Fatal("original tensor exceeds captured storage")
	}
	values := make([]float32, count)
	for i := range values {
		left, offset := int64(i), int64(0)
		for a := len(ref.Shape) - 1; a >= 0; a-- {
			offset += left % ref.Shape[a] * ref.Stride[a]
			left /= ref.Shape[a]
		}
		values[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[offset*4:]))
	}
	return values
}

// Every captured invocation is replayed with only its real waveform and power
// inputs. Native intermediate tensors are never injected into the Go processor.
func TestCompressorOriginal(t *testing.T) {
	config, err := os.ReadFile("../../assets/synth/compressor.json")
	if err != nil {
		t.Fatal(err)
	}
	configHash := fmt.Sprintf("%x", sha256.Sum256(config))
	var c Compressor
	if err = json.Unmarshal(config, &c); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		root, hash, baseline string
		calls, count         int
	}{
		{"compressor-operators-short", "5c84a5c1c0efd90f8fb90bfb331b8ad6c8f5380db3b74ee5604c5b5c13f93433", "2bd9a6674b3ac583fdeb88aca32814826b24d69882bfe6821d8fd87d7077738d", 1, 18000},
		{"compressor-operators-normal", "bdc4635b8941e88d859c8c4fa924adcc920f137397be98330185691150aa9ba7", "8211e40d691b2017dab22227d04026c5742df572d9499c430bcd9030f7ccfe56", 2, 39600},
		{"compressor-operators-tail-retry", "7a9f2731acfccad5ae8a46ff7cacc03603b72a7dab3090d03417a5ef56d8616b", "9e14c6655f3efe4af71985a5ebb4c13e1f3177090755cfd04958ddf60e28ecd5", 10, 228240},
	} {
		t.Run(tc.root, func(t *testing.T) {
			root := filepath.Join("..", "..", "private", tc.root)
			data, err := os.ReadFile(filepath.Join(root, "postprocessor.json"))
			if os.IsNotExist(err) {
				t.Skip("private original compressor capture unavailable")
			}
			if err != nil {
				t.Fatal(err)
			}
			var ref struct {
				ModelHash    string `json:"model_sha256"`
				DLLHash      string `json:"dll_sha256"`
				EngineHash   string `json:"engine_sha256"`
				ConfigHash   string `json:"config_sha256"`
				BaselineHash string `json:"baseline_sha256"`
				Hash         string `json:"output_sha256"`
				Threads      int
				Count        int `json:"value_count"`
				Calls        []struct {
					Run                 int
					Wave, Power, Output originalTensor
				}
			}
			if err = json.Unmarshal(data, &ref); err != nil {
				t.Fatal(err)
			}
			if ref.ModelHash != "40b90499c600b42b97b8937f56933db2ed5c8d328247de55ee2d24ce39dd280d" || ref.DLLHash != "00ca5c937940a94d052dbe52ec5fb0909db11ffbfccfd6e65e19b468169ff5e5" || ref.EngineHash != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || ref.ConfigHash != "cc6409b2162eaa00a3262d2518bb18d43a0187140410684860faacd2981d895e" || ref.ConfigHash != configHash || ref.BaselineHash != tc.baseline || ref.Hash != tc.hash || ref.Count != tc.count || len(ref.Calls) != tc.calls || ref.Threads != 6 {
				t.Fatal("original compressor provenance/count differs")
			}
			hash, total := sha256.New(), 0
			for _, call := range ref.Calls {
				if len(call.Power.Shape) != 3 || !slices.Equal(call.Power.Shape[:2], []int64{1, 1}) || !slices.Equal(call.Wave.Shape, []int64{1, 1, call.Power.Shape[2] * 240}) || !slices.Equal(call.Output.Shape, call.Wave.Shape) {
					t.Fatal("original compressor shape differs")
				}
				got, err := c.Process(readOriginalTensor(t, root, call.Wave), readOriginalTensor(t, root, call.Power))
				if err != nil {
					t.Fatal(err)
				}
				want := readOriginalTensor(t, root, call.Output)
				if len(got) != len(want) {
					t.Fatal("original compressor output size differs")
				}
				for i, value := range got {
					bits := math.Float32bits(value)
					if bits != math.Float32bits(want[i]) {
						t.Fatalf("run %d sample %d: %08x != %08x", call.Run, i, bits, math.Float32bits(want[i]))
					}
					hash.Write(binary.LittleEndian.AppendUint32(nil, bits))
				}
				total += len(got)
			}
			if total != tc.count || fmt.Sprintf("%x", hash.Sum(nil)) != tc.hash {
				t.Fatal("original compressor output digest differs")
			}
			t.Logf("Compared %d original compressor calls, %d values; bit_differences=0", tc.calls, total)
		})
	}
}
