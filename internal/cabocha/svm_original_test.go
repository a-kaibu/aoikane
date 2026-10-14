package cabocha

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestSVMOriginal(t *testing.T) {
	dir := filepath.Join("..", "..", "private", "cabocha")
	read := func(name, hash string) []byte {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(dir, name))
		if os.IsNotExist(err) {
			t.Skip("private original dependency model/reference unavailable")
		}
		if err != nil {
			t.Fatal(err)
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(b)); got != hash {
			t.Fatalf("%s hash %s", name, got)
		}
		return b
	}
	raw := read("standard-dep.ipa.model", "2380fc689e2b21f063869e5e1f8ae304b2d2bf5f75cad6a7421c9fa0dbd7cb4f")
	refs := read("svm-original.json", "33eb10f267d7b07191b7b086056517b270b8a82dfc9b8594cc80a15c0c0bbd06")
	var original struct {
		Calls []struct {
			IDs  []int  `json:"ids"`
			Bits string `json:"bits"`
		} `json:"calls"`
		Lookups []struct {
			Key     string `json:"key"`
			Encoded string `json:"encoded"`
			ID      int    `json:"id"`
		} `json:"lookups"`
	}
	if err := json.Unmarshal(refs, &original); err != nil {
		t.Fatal(err)
	}
	m, err := ParseSVM(raw)
	if err != nil {
		t.Fatal(err)
	}
	if m.FeatureCount() != 122541 || m.FrequentFeatureCount() != 3000 || m.Parameter("charset") != "SHIFT_JIS" || m.Parameter("degree") != "2" {
		t.Fatal("model metadata differs")
	}
	// Corrupt header dimensions must be rejected before allocations or slicing.
	for _, tc := range []struct {
		name   string
		offset int
		value  uint32
	}{
		{"version", 4, 101}, {"parameter-length", 8, 0xffffffff},
		{"normalization-nan", 132, 0x7fc00000}, {"normalization-inf", 132, 0x7f800000},
		{"normalization-zero", 132, 0}, {"normalization-negative", 132, 0xbf800000},
		{"feature-count", 140, 0xffffffff}, {"frequent-count", 144, 0xffffffff},
		{"empty-dictionary", 148, 0}, {"unaligned-pairs", 152, 1},
		{"bad-node", 156 + 1801216 + 22100992, 0xffffffff},
	} {
		old := binary.LittleEndian.Uint32(raw[tc.offset:])
		binary.LittleEndian.PutUint32(raw[tc.offset:], tc.value)
		_, err := ParseSVM(raw)
		binary.LittleEndian.PutUint32(raw[tc.offset:], old)
		if err == nil {
			t.Fatalf("accepted malformed %s", tc.name)
		}
	}
	for _, size := range []int{0, 1, 35, 155, len(raw) - 1} {
		truncated := raw[:size]
		if size >= 4 {
			binary.LittleEndian.PutUint32(raw, uint32(size)^0xef522177)
		}
		_, err := ParseSVM(truncated)
		binary.LittleEndian.PutUint32(raw, uint32(len(raw))^0xef522177)
		if err == nil {
			t.Fatalf("accepted truncation to %d", size)
		}
	}
	// Prove that the model owns the arrays rather than aliasing its input.
	clear(raw)
	for i, call := range original.Calls {
		want, err := hex.DecodeString(call.Bits)
		if err != nil || len(want) != 8 {
			t.Fatalf("reference %d: invalid bits", i)
		}
		got, err := m.Score(call.IDs)
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		if math.Float64bits(got) != binary.LittleEndian.Uint64(want) {
			t.Fatalf("call %d (%d IDs): got %.17g (%016x), want %x", i, len(call.IDs), got, math.Float64bits(got), want)
		}
	}
	// Model strings are CP932 bytes. The codec is supplied by the frontend;
	// original lookup cases include Japanese surface and POS feature strings.
	positives := 0
	for _, lookup := range original.Lookups {
		encoded, err := hex.DecodeString(lookup.Encoded)
		if err != nil {
			t.Fatal(err)
		}
		got, err := m.ID(encoded)
		if err != nil || got != lookup.ID {
			t.Fatalf("lookup %q: got %d, %v; want %d", lookup.Key, got, err, lookup.ID)
		}
		if got >= 0 {
			positives++
		}
	}
	if len(original.Calls) != 262 || len(original.Lookups) != 207 || positives < 50 {
		t.Fatal("incomplete oracle comparisons")
	}
	for _, ids := range [][]int{nil, {3000}, {-1}, {0, 0}, {1, 0}, {0, 122541}} {
		if _, err := m.Score(ids); err == nil {
			t.Fatalf("accepted invalid IDs %v", ids)
		}
	}
	t.Logf("%d original scores match all float64 bits; %d lookups (%d present) match", len(original.Calls), len(original.Lookups), positives)
}

// These vectors and sums were observed during ordinary Akane synthesis, rather
// than supplied to a reconstructed C++ object. The observation tool separately
// verifies that its WAV is identical to an uninstrumented original invocation.
func TestSVMTalkOriginal(t *testing.T) {
	read := func(path, hash string) []byte {
		t.Helper()
		b, err := os.ReadFile(filepath.Join("..", "..", "private", filepath.FromSlash(path)))
		if os.IsNotExist(err) {
			t.Skip("private original SVM talk reference unavailable")
		}
		if err != nil {
			t.Fatal(err)
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(b)); got != hash {
			t.Fatalf("%s hash %s", path, got)
		}
		return b
	}
	raw := read("cabocha/standard-dep.ipa.model", "2380fc689e2b21f063869e5e1f8ae304b2d2bf5f75cad6a7421c9fa0dbd7cb4f")
	data := read("cabocha-talk-akane/svm-talk.json", "881a43e9a9a5bbe8ccf5d13c116a97a14965eff96fd83c1e5c3f06c953c9b19f")
	var ref struct {
		Calls []struct {
			IDs         []int  `json:"ids"`
			Accumulator int32  `json:"accumulator"`
			Bits        string `json:"bits"`
		} `json:"calls"`
		WAVHash string `json:"wav_sha256"`
	}
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	if len(ref.Calls) != 5 || ref.WAVHash != "66ad8e91dee4a8c12e610a369db335b26e9a1fcd388fb38fb68a97e76fc1399a" {
		t.Fatal("incomplete normal-talk observation")
	}
	for _, name := range []string{"baseline.wav", "observed.wav"} {
		read("cabocha-talk-akane/"+name, ref.WAVHash)
	}
	m, err := ParseSVM(raw)
	if err != nil {
		t.Fatal(err)
	}
	for i, call := range ref.Calls {
		sum, err := m.scoreSum(call.IDs)
		if err != nil || sum != call.Accumulator {
			t.Fatalf("call %d: sum %d, %v; want %d", i, sum, err, call.Accumulator)
		}
		score, err := m.Score(call.IDs)
		bits, decodeErr := hex.DecodeString(call.Bits)
		if err != nil || decodeErr != nil || len(bits) != 8 || math.Float64bits(score) != binary.LittleEndian.Uint64(bits) {
			t.Fatalf("call %d: normalized score differs", i)
		}
	}
	t.Logf("%d normal-talk integer sums and normalized scores match original; observed and independent WAVs match", len(ref.Calls))
}
