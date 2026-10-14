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

func TestMKLTanhOriginal(t *testing.T) {
	data, err := os.ReadFile("testdata/mkl_tanh.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		DLLHash string `json:"dll_sha256"`
		RVA     string
		Worker  string `json:"worker_rva"`
		Mode    int
		Hash    string `json:"output_sha256"`
		Cases   []struct {
			Input  []uint32 `json:"input_bits"`
			Output []uint32 `json:"output_bits"`
		}
	}
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.DLLHash != "00ca5c937940a94d052dbe52ec5fb0909db11ffbfccfd6e65e19b468169ff5e5" || ref.RVA != "0x33020" || ref.Worker != "0x5bc300" || ref.Mode != 0x140102 || len(ref.Cases) != 45 {
		t.Fatal("original Tanh provenance/count differs")
	}
	hash, total := sha256.New(), 0
	for ci, c := range ref.Cases {
		if len(c.Input) != len(c.Output) {
			t.Fatal("invalid original Tanh reference")
		}
		for i, bits := range c.Input {
			got := math.Float32bits(MKLTanh(math.Float32frombits(bits)))
			if got != c.Output[i] {
				t.Fatalf("case %d input %d=%08x: got %08x want %08x", ci, i, bits, got, c.Output[i])
			}
			hash.Write(binary.LittleEndian.AppendUint32(nil, c.Output[i]))
			total++
		}
	}
	t.Logf("Compared %d original Tanh values; SHA256 %x", total, hash.Sum(nil))
	if total != 139794 || ref.Hash != "7ca82d23a2ed598216ca6d1afc4f02c1d64f02dc5be197f9eab819aec878e5b6" || fmt.Sprintf("%x", hash.Sum(nil)) != ref.Hash {
		t.Fatal("original Tanh reference hash differs")
	}
}
