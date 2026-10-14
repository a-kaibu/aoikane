// Compare Go GEMM with individual kernel outputs observed in the original engine.
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/a-kaibu/aoikane/internal/onnx"
)

type tensor struct {
	ID          int   `json:"id"`
	Shape       []int `json:"shape"`
	Stride      []int `json:"stride"`
	DType       int   `json:"dtype"`
	Alignment64 int   `json:"alignment64"`
}

type event struct {
	Backend   string  `json:"backend"`
	Kind      string  `json:"kind"`
	Event     string  `json:"event"`
	Operation int     `json:"operation"`
	Left      tensor  `json:"left"`
	Right     tensor  `json:"right"`
	Output    tensor  `json:"output"`
	Alpha     float32 `json:"alpha"`
	Skipped   string  `json:"value_capture_skipped"`
	TransB    string  `json:"trans_b"`
	TransA    string  `json:"trans_a"`
	Beta      float32 `json:"beta"`
	Initial   tensor  `json:"initial"`
}

func read(root string, t tensor) ([]float32, error) {
	if t.ID < 1 || t.DType != 6 || len(t.Shape) != len(t.Stride) || len(t.Shape) == 0 {
		return nil, fmt.Errorf("invalid FLOAT tensor metadata")
	}
	span := 1
	for i, size := range t.Shape {
		if size < 1 || t.Stride[i] < 1 {
			return nil, fmt.Errorf("unsupported tensor shape/stride")
		}
		span += (size - 1) * t.Stride[i]
	}
	data, err := os.ReadFile(filepath.Join(root, fmt.Sprintf("model-%d.bin", t.ID)))
	if err != nil {
		return nil, err
	}
	if len(data) != 4*span {
		return nil, fmt.Errorf("tensor %d has %d bytes, want %d", t.ID, len(data), 4*span)
	}
	values := make([]float32, span)
	for i := range values {
		values[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[4*i:]))
	}
	return values, nil
}

func run(root string) error {
	data, err := os.ReadFile(filepath.Join(root, "events.json"))
	if err != nil {
		return err
	}
	var events []event
	if err := json.Unmarshal(data, &events); err != nil {
		return err
	}
	type count struct{ calls, values, differences int }
	counts := map[int]count{}
	total := count{}
	skipped := 0
	hash := sha256.New()
	for _, e := range events {
		if e.Kind == "gemm" && e.Skipped != "" {
			skipped++
		}
		if e.Event != "torch_gemm_result" {
			continue
		}
		if e.Backend != "" && e.Backend != "dnnl" {
			return fmt.Errorf("unsupported original GEMM backend %q", e.Backend)
		}
		if e.Backend == "dnnl" && (e.TransA != "N" || (e.TransB != "N" && e.TransB != "T")) {
			return fmt.Errorf("unsupported original oneDNN transpose")
		}
		if len(e.Left.Shape) != 2 || len(e.Left.Stride) != 2 || len(e.Right.Shape) != 2 || len(e.Right.Stride) != 2 || len(e.Output.Shape) != 2 || len(e.Output.Stride) != 2 || e.Left.Stride[1] != 1 || e.Right.Stride[1] != 1 || e.Output.Stride[1] != 1 {
			return fmt.Errorf("operation %d: incompatible GEMM metadata", e.Operation)
		}
		m, n, k := e.Left.Shape[0], e.Output.Shape[0], e.Left.Shape[1]
		if e.TransA == "N" {
			m, k = e.Left.Shape[1], e.Left.Shape[0]
		} else if e.TransA != "" && e.TransA != "T" {
			return fmt.Errorf("unsupported original GEMM transpose A %q", e.TransA)
		}
		if m != e.Output.Shape[1] || (e.TransA != "N" && e.Beta != 0) {
			return fmt.Errorf("operation %d: incompatible GEMM dimensions/transpose/beta", e.Operation)
		}
		if e.TransB == "T" {
			if e.Right.Shape[0] != k || e.Right.Shape[1] != n {
				return fmt.Errorf("operation %d: incompatible T,T GEMM metadata", e.Operation)
			}
		} else if e.TransB == "N" || e.TransB == "" { // Older T,N captures omit this field.
			if e.Right.Shape[0] != n || e.Right.Shape[1] != k {
				return fmt.Errorf("operation %d: incompatible T,N GEMM metadata", e.Operation)
			}
		} else {
			return fmt.Errorf("unsupported original GEMM transpose %q", e.TransB)
		}
		a, err := read(root, e.Left)
		if err != nil {
			return err
		}
		x, err := read(root, e.Right)
		if err != nil {
			return err
		}
		want, err := read(root, e.Output)
		if err != nil {
			return err
		}
		var y []float32
		if e.TransA == "N" {
			var initial []float32
			if e.Beta != 0 {
				if len(e.Initial.Shape) != 2 || len(e.Initial.Stride) != 2 || e.Initial.Shape[0] != n || e.Initial.Shape[1] != m || e.Initial.Stride[0] != e.Output.Stride[0] || e.Initial.Stride[1] != 1 {
					return fmt.Errorf("operation %d: incompatible initial GEMM metadata", e.Operation)
				}
				initial, err = read(root, e.Initial)
				if err != nil {
					return err
				}
			}
			if e.Backend == "dnnl" {
				if e.Alpha != 1 {
					return fmt.Errorf("unsupported original oneDNN alpha")
				}
				if e.TransB == "T" {
					y, err = onnx.DNNLGEMMNormalT(a, x, initial, m, n, k, e.Left.Stride[0], e.Right.Stride[0], e.Output.Stride[0], e.Beta)
				} else {
					y, err = onnx.DNNLGEMMNormalN(a, x, initial, m, n, k, e.Left.Stride[0], e.Right.Stride[0], e.Output.Stride[0], e.Beta)
				}
			} else if e.TransB == "T" {
				y, err = onnx.MKLGEMMNormalT(a, x, initial, m, n, k, e.Left.Stride[0], e.Right.Stride[0], e.Output.Stride[0], e.Output.Alignment64%16, e.Alpha, e.Beta)
			} else {
				y, err = onnx.MKLGEMMNormalN(a, x, initial, m, n, k, e.Left.Stride[0], e.Right.Stride[0], e.Output.Stride[0], e.Output.Alignment64%16, e.Alpha, e.Beta)
			}
		} else if e.TransB == "T" {
			y, err = onnx.MKLGEMMTransposeT(a, x, m, n, k, e.Left.Stride[0], e.Right.Stride[0], e.Alpha)
		} else {
			y, err = onnx.MKLGEMMTransposeN(a, x, m, n, k, e.Left.Stride[0], e.Right.Stride[0], e.Right.Alignment64%16, e.Alpha)
		}
		if err != nil {
			return err
		}
		c := counts[e.Operation]
		c.calls++
		c.values += len(y)
		for i, value := range y {
			bits := math.Float32bits(want[(i/m)*e.Output.Stride[0]+i%m])
			var bytes [4]byte
			binary.LittleEndian.PutUint32(bytes[:], bits)
			hash.Write(bytes[:])
			if math.Float32bits(value) != bits {
				c.differences++
			}
		}
		counts[e.Operation] = c
		total.calls++
		total.values += len(y)
	}
	for op := 1; op <= len(events); op++ {
		if c, ok := counts[op]; ok {
			fmt.Printf("operation %d: %d calls, %d values, %d bit differences\n", op, c.calls, c.values, c.differences)
			total.differences += c.differences
		}
	}
	if total.calls == 0 {
		return fmt.Errorf("no original GEMM results found")
	}
	fmt.Printf("total: %d calls, %d values, %d bit differences; original SHA256 %x\n", total.calls, total.values, total.differences, hash.Sum(nil))
	if skipped != 0 {
		fmt.Printf("not compared: %d GEMM calls without captured values (geometry only)\n", skipped)
	}
	if total.differences != 0 {
		return fmt.Errorf("original GEMM differs")
	}
	return nil
}

func main() {
	root := flag.String("trace", "", "original --torch-gemm-values observation directory")
	flag.Parse()
	if *root == "" {
		fmt.Fprintln(os.Stderr, "-trace is required")
		os.Exit(2)
	}
	if err := run(*root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
