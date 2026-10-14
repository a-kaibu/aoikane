package onnx

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestOriginalLinearCaptured(t *testing.T) {
	for _, c := range []struct {
		trace  string
		values int
		hash   string
	}{
		{"torch-conv-all-short", 18145, "cfe44410159c24de1b6f4b9b5cbc51b8f94cb1277be9e558b5f578dc149d988b"},
		{"featnet-short-exact", 7353, "6556a4b38dad0c9ea3e55285a1882fb5dc6ff737909ba3494ecc89d9ea912aa4"},
		{"featnet-four-phones", 9938, "ee075d45348ade3a77521f00b106e0fd694cab32ef6f1546e1a63867c1045b44"},
		{"torch-conv-all-medium", 42359, "d62a8f761b5fb0666ca805322b092fd5b4ae953f2640fa6ce94643019b827626"},
		{"torch-conv-all-long", 535591, "4aa8f07e1e58d2e199fc7812cec0584c3bb661437718a7818f988d2c244b9c8f"},
	} {
		t.Run(c.trace, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "..", "private", c.trace, "linears.json"))
			if os.IsNotExist(err) {
				t.Skip("private original engine linear capture unavailable")
			}
			if err != nil {
				t.Fatal(err)
			}
			var records []struct {
				Node, Op            string
				InputShape          []int64 `json:"input_shape"`
				OutputShape         []int64 `json:"output_shape"`
				Input, Output, Bias []float32
			}
			if err := json.Unmarshal(data, &records); err != nil {
				t.Fatal(err)
			}
			if len(records) != 13 {
				t.Fatal("original linear operation count differs")
			}
			hash, total := sha256.New(), 0
			seen := map[string]bool{}
			for _, r := range records {
				if r.Op != "MatMul" || seen[r.Node] || len(r.InputShape) != 3 || r.InputShape[0] != 1 || len(r.OutputShape) != 3 || r.OutputShape[0] != 1 {
					t.Fatal("invalid original linear capture")
				}
				seen[r.Node] = true
				l, err := OpenLinear("../../assets/synth/featnet.onnx", r.Node)
				if err != nil {
					t.Fatal(err)
				}
				y, err := l.RunOriginalLinear(r.Input, int(r.InputShape[1]))
				if err != nil {
					t.Fatal(err)
				}
				if len(y) != len(r.Output) || r.OutputShape[1] != r.InputShape[1] || r.OutputShape[2] != int64(l.Out) || (len(r.Bias) != 0 && len(r.Bias) != l.Out) {
					t.Fatal("original linear shape/bias differs")
				}
				for i, value := range y {
					if len(r.Bias) != 0 {
						value = float32(value + r.Bias[i%l.Out])
					}
					bits, want := math.Float32bits(value), math.Float32bits(r.Output[i])
					if bits != want {
						t.Fatalf("%s output %d: got %08x want %08x", r.Node, i, bits, want)
					}
					hash.Write(binary.LittleEndian.AppendUint32(nil, want))
				}
				total += len(y)
			}
			if total != c.values || fmt.Sprintf("%x", hash.Sum(nil)) != c.hash {
				t.Fatalf("original linear count/hash differs: %d %x", total, hash.Sum(nil))
			}
		})
	}
}
