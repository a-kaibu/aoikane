package onnx

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// Original captures and model weights stay private. This test uses only Go;
// it never loads PyTorch, oneDNN, MKL, ONNX Runtime or any shared library.
func TestOriginalConvCaptured(t *testing.T) {
	for _, capture := range []struct {
		trace, file   string
		values, count int
		hash          string
	}{
		{"torch-conv-all-short", "operations.json", 2372640, 100, "15a157e2d0f00e9c1862e9dee276a3cbb6ebc4661d96443a540dcf3116888125"},
		{"torch-conv-all-second", "operations.json", 3032160, 100, "083c2ab37b52d1f7ec70abae7f01f3231f3272080fb881819869fbb12adfa21a"},
		{"torch-conv-all-medium", "operations.json", 5923360, 100, "3c64d793c156af60ee90a4da8140d867f023a24ac44fd1acb0b0ee9f37f3e81c"},
		{"torch-conv-all-long", "phoneme-operations.json", 3839520, 55, "bbadac5458f3ef9cf5e2e638e958e7b7021a7f277064d668b02842517e80caa8"},
	} {
		t.Run(capture.trace, func(t *testing.T) {
			file, err := os.Open(filepath.Join("..", "..", "private", capture.trace, capture.file))
			if os.IsNotExist(err) {
				t.Skip("private original engine capture unavailable")
			}
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			decoder := json.NewDecoder(file)
			if token, err := decoder.Token(); err != nil || token != json.Delim('[') {
				t.Fatal("invalid original Conv capture array")
			}
			hash := sha256.New()
			total, count := 0, 0
			for decoder.More() {
				var r struct {
					Op, Node      string
					InputShape    []int64 `json:"input_shape"`
					OutputShape   []int64 `json:"output_shape"`
					Input, Output []float32
				}
				if err := decoder.Decode(&r); err != nil {
					t.Fatal(err)
				}
				if r.Op != "Conv" {
					continue
				}
				if len(r.InputShape) != 3 || r.InputShape[0] != 1 || len(r.OutputShape) != 3 || r.OutputShape[0] != 1 {
					t.Fatal("invalid captured Conv shape")
				}
				c, err := OpenConv1D(filepath.Join("..", "..", "assets", "synth", "featnet.onnx"), r.Node)
				if err != nil {
					t.Fatal(err)
				}
				y, err := c.RunOriginal(r.Input, int(r.InputShape[2]))
				if err != nil {
					t.Fatal(err)
				}
				if len(y) != len(r.Output) || r.OutputShape[1] != int64(c.Out) || r.OutputShape[2] != int64(len(y)/c.Out) {
					t.Fatal("captured Conv output size differs")
				}
				for i, value := range y {
					want := math.Float32bits(r.Output[i])
					if bits := math.Float32bits(value); bits != want {
						t.Fatalf("%s output %d: got %08x want %08x", r.Node, i, bits, want)
					}
					hash.Write(binary.LittleEndian.AppendUint32(nil, want))
				}
				total += len(y)
				count++
			}
			if token, err := decoder.Token(); err != nil || token != json.Delim(']') {
				t.Fatal("invalid original Conv capture array end")
			}
			if err := decoder.Decode(new(any)); err != io.EOF {
				t.Fatal("unexpected data after original Conv capture")
			}
			if total != capture.values || count != capture.count || fmt.Sprintf("%x", hash.Sum(nil)) != capture.hash {
				t.Fatal("original Conv reference count/hash differs")
			}
		})
	}
}
