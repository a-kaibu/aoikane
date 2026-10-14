package synthesis

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

// Exercise the production model loader and vocoder adapter against original
// native waveforms. An absent runtime path must work without a native library.
func TestVocoderOriginal(t *testing.T) {
	for _, tc := range []struct {
		root, hash string
		frames     int
	}{
		{"vocoder-all-conv-native", "52b6151ac4b02d6a5144b6d19beedb38e45f7c0c4250f25e80ff83a70c1f8238", 110},
		{"vocoder-transpose-all-short", "f0e47b656610fa8375abf26a314694d975043137c84039e4704f7a253b127300", 75},
		{"vocoder-transpose-all-tail", "e8e4738a57c3b2999f50ee576e89ffc4aeb59d3e64efe0ba31643cb905ac173d", 55},
	} {
		t.Run(tc.root, func(t *testing.T) {
			root := filepath.Join("..", "..", "private", tc.root)
			data, err := os.ReadFile(filepath.Join(root, "operators.json"))
			if os.IsNotExist(err) {
				t.Skip("private original vocoder capture unavailable")
			}
			if err != nil {
				t.Fatal(err)
			}
			var ref struct {
				Inputs map[string]capturedTensor
				Wave   capturedTensor
				Hash   string `json:"native_wave_sha256"`
			}
			if err = json.Unmarshal(data, &ref); err != nil {
				t.Fatal(err)
			}
			if ref.Hash != tc.hash || len(ref.Inputs) != 3 {
				t.Fatal("original vocoder reference differs")
			}
			e, err := Open(filepath.Join(t.TempDir(), "missing-runtime"), filepath.Join("..", "..", "assets", "synth"))
			if err != nil {
				t.Fatal(err)
			}
			defer e.Close()
			var inputs []modelTensor
			for _, name := range []string{"f0", "bap", "spc"} {
				r := ref.Inputs[name]
				inputs = append(inputs, modelTensor{Name: name, Type: 1, Shape: r.Shape, Data: capturedBytes(t, root, r)})
			}
			out, err := runOriginalModel(e.vocoder, inputs, []string{"wave"})
			if err != nil {
				t.Fatal(err)
			}
			shape := []int64{1, 1, int64(tc.frames * 240)}
			if len(out) != 1 || out[0].Name != "wave" || !slices.Equal(out[0].Shape, shape) || !slices.Equal(ref.Wave.Shape, shape) || len(out[0].Data) != tc.frames*240 {
				t.Fatal("original wave shape/count differs")
			}
			want := capturedBytes(t, root, ref.Wave)
			got := make([]byte, len(out[0].Data)*4)
			if len(got) != len(want) {
				t.Fatal("original wave storage size differs")
			}
			for i, x := range out[0].Data {
				bits := math.Float32bits(x)
				w := binary.LittleEndian.Uint32(want[i*4:])
				if bits != w {
					t.Fatalf("wave sample %d: %08x != %08x", i, bits, w)
				}
				binary.LittleEndian.PutUint32(got[i*4:], bits)
			}
			if fmt.Sprintf("%x", sha256.Sum256(got)) != tc.hash {
				t.Fatal("original wave hash differs")
			}
			t.Logf("Production pure-Go vocoder: %d values, bit_differences=0", tc.frames*240)
		})
	}
}
