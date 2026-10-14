// Compare Go kernels with operation outputs observed inside the original engine.
package main

import (
	"github.com/a-kaibu/aoikane/internal/onnx"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"slices"
	"strings"
)

type record struct {
	Operation   int       `json:"operation"`
	Node        string    `json:"node"`
	Op          string    `json:"op"`
	InputShape  []int64   `json:"input_shape"`
	Input       []float32 `json:"input"`
	Bias        []float32 `json:"bias"`
	OutputShape []int64   `json:"output_shape"`
	Output      []float32 `json:"output"`
}

func run() error {
	path := flag.String("inputs", "", "export_torch_operations.py output JSON")
	model := flag.String("model", "assets/synth/featnet.onnx", "original model weights")
	nodes := flag.String("nodes", "", "comma-separated exact node names to compare (empty selects all)")
	originalEmbedding := flag.Bool("original-embedding", false, "use audited original MKL art/nima/niab embedding kernels")
	originalLinear := flag.Bool("original-linear", false, "use original MKL kernels for all FeatNet linear projections")
	originalConv := flag.Bool("original-conv", false, "use audited original MKL/oneDNN convolution kernels")
	convOnly := flag.Bool("conv-only", false, "compare only captured Conv operations")
	linearOnly := flag.Bool("linear-only", false, "compare only captured MatMul operations")
	flag.Parse()
	if *convOnly && *linearOnly {
		return fmt.Errorf("-conv-only and -linear-only are mutually exclusive")
	}
	selected := map[string]bool{}
	if *nodes != "" {
		for _, node := range strings.Split(*nodes, ",") {
			if node == "" || selected[node] {
				return fmt.Errorf("empty or duplicate selected node")
			}
			selected[node] = true
		}
	}
	inputFile, err := os.Open(*path)
	if err != nil {
		return err
	}
	defer inputFile.Close()
	decoder := json.NewDecoder(inputFile)
	first, err := decoder.Token()
	if err != nil {
		return err
	}
	if first != json.Delim('[') {
		return fmt.Errorf("expected operation array")
	}
	if !decoder.More() {
		return fmt.Errorf("no captured operations")
	}
	bad, total := 0, 0
	hash := sha256.New()
	seen := map[string]bool{}
	for decoder.More() {
		var r record
		if err := decoder.Decode(&r); err != nil {
			return err
		}
		if *convOnly && r.Op != "Conv" {
			continue
		}
		if *linearOnly && r.Op != "MatMul" {
			continue
		}
		if len(selected) != 0 && !selected[r.Node] {
			continue
		}
		seen[r.Node] = true
		if len(r.InputShape) != 3 || r.InputShape[0] != 1 || r.InputShape[1] <= 0 || r.InputShape[2] <= 0 {
			return fmt.Errorf("invalid native input shape")
		}
		var got []float32
		var shape []int64
		switch r.Op {
		case "MatMul":
			l, err := onnx.OpenLinear(*model, r.Node)
			if err != nil {
				return err
			}
			if *originalLinear {
				got, err = l.RunOriginalLinear(r.Input, int(r.InputShape[1]))
			} else if *originalEmbedding {
				got, err = l.RunOriginalEmbedding(r.Input, int(r.InputShape[1]))
			} else {
				got, err = l.Run(r.Input, int(r.InputShape[1]))
			}
			if err != nil {
				return err
			}
			shape = []int64{1, r.InputShape[1], int64(l.Out)}
			if len(r.Bias) != 0 {
				if len(r.Bias) != l.Out {
					return fmt.Errorf("invalid native linear bias")
				}
				for i := range got {
					got[i] = float32(got[i] + r.Bias[i%l.Out])
				}
			}
		case "Conv":
			if *originalEmbedding && !*originalConv {
				return fmt.Errorf("original embedding backend does not support Conv")
			}
			c, err := onnx.OpenConv1D(*model, r.Node)
			if err != nil {
				return err
			}
			if *originalConv {
				got, err = c.RunOriginal(r.Input, int(r.InputShape[2]))
			} else {
				got, err = c.Run(r.Input, int(r.InputShape[2]))
			}
			if err != nil {
				return err
			}
			shape = []int64{1, int64(c.Out), int64(len(got) / c.Out)}
		default:
			return fmt.Errorf("unsupported observed operation %s", r.Op)
		}
		if !slices.Equal(shape, r.OutputShape) || len(got) != len(r.Output) {
			return fmt.Errorf("operation %d shape differs", r.Operation)
		}
		diffs, maxError := 0, float64(0)
		for i, v := range got {
			w := r.Output[i]
			hash.Write(binary.LittleEndian.AppendUint32(nil, math.Float32bits(w)))
			if math.Float32bits(v) != math.Float32bits(w) {
				diffs++
			}
			maxError = max(maxError, math.Abs(float64(v)-float64(w)))
		}
		fmt.Printf("native operation=%d node=%s values=%d bit_differences=%d max_error=%.9g\n", r.Operation, r.Node, len(got), diffs, maxError)
		bad += diffs
		total += len(got)
	}
	if _, err := decoder.Token(); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("unexpected data after operation array: %v", err)
	}
	for node := range selected {
		if !seen[node] {
			return fmt.Errorf("selected node %q not found", node)
		}
	}
	if total == 0 {
		return fmt.Errorf("no selected original engine values")
	}
	fmt.Printf("Compared %d captured original engine values; %d bit differences; original SHA256 %x\n", total, bad, hash.Sum(nil))
	if bad != 0 {
		return fmt.Errorf("Go kernels differ from original engine")
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
