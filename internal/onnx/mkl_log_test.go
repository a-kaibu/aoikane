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

func TestMKLLogOriginal(t *testing.T) {
	data, err := os.ReadFile("testdata/mkl_log.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		DLLSHA256      string `json:"dll_sha256"`
		RVA            string
		WorkerRVA      string `json:"worker_rva"`
		Mode           int
		Hash           string `json:"output_sha256"`
		SubnormalCount int    `json:"subnormal_count"`
		SubnormalHash  string `json:"subnormal_sha256"`
		Cases          []struct {
			Input  []uint32 `json:"input_bits"`
			Output []uint32 `json:"output_bits"`
		}
	}
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.DLLSHA256 != "00ca5c937940a94d052dbe52ec5fb0909db11ffbfccfd6e65e19b468169ff5e5" || ref.RVA != "0x31b10" || ref.WorkerRVA != "0x43b420" || ref.Mode != 0x140102 || len(ref.Cases) != 40 {
		t.Fatal("unexpected original Log provenance/count")
	}
	hash, total := sha256.New(), 0
	for ci, c := range ref.Cases {
		if len(c.Input) != len(c.Output) {
			t.Fatal("invalid original Log reference")
		}
		for i, bits := range c.Input {
			got, want := math.Float32bits(MKLLog(math.Float32frombits(bits))), c.Output[i]
			if got != want {
				t.Fatalf("case %d value %d input=%08x: got %08x want %08x", ci, i, bits, got, want)
			}
			hash.Write(binary.LittleEndian.AppendUint32(nil, want))
			total++
		}
	}
	if total != 33548 || ref.Hash != "60e1d36f51d6c32915e46ad63c5277f86674c33abce41b250a8825c44fd18e33" || fmt.Sprintf("%x", hash.Sum(nil)) != ref.Hash {
		t.Fatal("original Log reference count/hash differs")
	}
	if ref.SubnormalCount != 0x7fffff || ref.SubnormalHash != "293c63f89abd017cf2991132f73804181e34250d8fb3f5ff1c75fd4a258ba1ad" {
		t.Fatal("original subnormal reference provenance differs")
	}
	hash.Reset()
	buffer := make([]byte, 65536*4)
	for start := uint32(1); start < 0x800000; start += 65536 {
		count := min(uint32(65536), 0x800000-start)
		for i := uint32(0); i < count; i++ {
			binary.LittleEndian.PutUint32(buffer[i*4:], math.Float32bits(MKLLog(math.Float32frombits(start+i))))
		}
		hash.Write(buffer[:count*4])
	}
	if got := fmt.Sprintf("%x", hash.Sum(nil)); got != ref.SubnormalHash {
		t.Fatalf("exhaustive subnormal output hash: got %s want %s", got, ref.SubnormalHash)
	}
}

func TestMKLLogCaptured(t *testing.T) {
	data, err := os.ReadFile("../../private/torch-controls-f0/unary.json")
	if os.IsNotExist(err) {
		t.Skip("private original Log capture unavailable")
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
		if r.Op != "log" {
			continue
		}
		count++
		if len(r.Input) != len(r.Output) {
			t.Fatal("invalid captured Log shape")
		}
		for i, x := range r.Input {
			got, want := math.Float32bits(MKLLog(x)), math.Float32bits(r.Output[i])
			if got != want {
				t.Fatalf("captured Log %d input=%g: got %08x want %08x", i, x, got, want)
			}
			hash.Write(binary.LittleEndian.AppendUint32(nil, want))
			total++
		}
	}
	if count != 1 || total != 10 || fmt.Sprintf("%x", hash.Sum(nil)) != "d5020149f5ed3586c1c32779f399189ef0ff337d6aa1608079913521dc85b9a3" {
		t.Fatal("captured original Log count/hash differs")
	}
}
