package onnx

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"testing"
)

func TestMKLExpOriginal(t *testing.T) {
	data, err := os.ReadFile("testdata/mkl_exp.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		DLLSHA256 string `json:"dll_sha256"`
		RVA       string
		WorkerRVA string `json:"worker_rva"`
		Mode      int
		Hash      string `json:"output_sha256"`
		Cases     []struct{ Input, Output []float32 }
		Special   struct {
			InputBits  []uint32 `json:"input_bits"`
			OutputBits []uint32 `json:"output_bits"`
		}
	}
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.DLLSHA256 != "00ca5c937940a94d052dbe52ec5fb0909db11ffbfccfd6e65e19b468169ff5e5" || ref.RVA != "0x31640" || ref.WorkerRVA != "0x41ac20" || ref.Mode != 0x140102 || len(ref.Cases) != 44 {
		t.Fatal("unexpected original MKL Exp provenance/count")
	}
	hash, total := sha256.New(), 0
	for ci, c := range ref.Cases {
		if len(c.Input) != len(c.Output) {
			t.Fatal("invalid original Exp reference")
		}
		for i, x := range c.Input {
			y := MKLExp(x)
			want := math.Float32bits(c.Output[i])
			hash.Write(binary.LittleEndian.AppendUint32(nil, want))
			if bits := math.Float32bits(y); bits != want {
				t.Fatalf("case %d value %d x=%g: got %08x want %08x", ci, i, x, bits, want)
			}
			total++
		}
	}
	if len(ref.Special.InputBits) != 4174 || len(ref.Special.OutputBits) != 4174 {
		t.Fatal("original exceptional Exp count differs")
	}
	for i, bits := range ref.Special.InputBits {
		y := MKLExp(math.Float32frombits(bits))
		want := ref.Special.OutputBits[i]
		hash.Write(binary.LittleEndian.AppendUint32(nil, want))
		if got := math.Float32bits(y); got != want {
			t.Fatalf("special %d input=%08x: got %08x want %08x", i, bits, got, want)
		}
		total++
	}
	if total != 21810 || ref.Hash != "fb922d8c2503ea5c0fc5e6d579a74f554dff3dcbe877fbb1d229cd0cbde8a645" || fmt.Sprintf("%x", hash.Sum(nil)) != ref.Hash {
		t.Fatal("original MKL Exp reference hash/count differs")
	}
}

func TestMKLExpCaptured(t *testing.T) {
	data, err := os.ReadFile("../../private/torch-unary-structured/unary.json")
	if os.IsNotExist(err) {
		t.Skip("private original unary capture unavailable")
	}
	if err != nil {
		t.Fatal(err)
	}
	var records []struct {
		Op            string
		Input, Output []float32
	}
	if err := json.Unmarshal(data, &records); err != nil {
		t.Fatal(err)
	}
	total, count := 0, 0
	hash := sha256.New()
	for _, r := range records {
		if r.Op != "exp" {
			continue
		}
		count++
		if len(r.Input) != len(r.Output) {
			t.Fatal("invalid captured Exp shape")
		}
		for i, x := range r.Input {
			y := MKLExp(x)
			if math.Float32bits(y) != math.Float32bits(r.Output[i]) {
				t.Fatalf("captured Exp %d: %g got %08x want %08x", i, x, math.Float32bits(y), math.Float32bits(r.Output[i]))
			}
			hash.Write(binary.LittleEndian.AppendUint32(nil, math.Float32bits(r.Output[i])))
			total++
		}
	}
	if count != 4 || total != 185 || fmt.Sprintf("%x", hash.Sum(nil)) != "dee2400b32d364fdfeb3b545828a7897b8dc7af119f234caf207a5069641e88c" {
		t.Fatal("captured original Exp count differs")
	}
}
