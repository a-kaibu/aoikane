// Compare Go GEMV with individual kernel outputs observed in the original engine.
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
	Kind      string  `json:"kind"`
	Event     string  `json:"event"`
	Operation int     `json:"operation"`
	Matrix    tensor  `json:"matrix"`
	Vector    tensor  `json:"vector"`
	Output    tensor  `json:"output"`
	Alpha     float32 `json:"alpha"`
	Skipped   string  `json:"value_capture_skipped"`
	Trans     string  `json:"trans"`
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
		if e.Kind == "gemv" && e.Skipped != "" {
			skipped++
		}
		if e.Event != "torch_gemv_result" {
			continue
		}
		if len(e.Matrix.Shape) != 2 || len(e.Matrix.Stride) != 2 || len(e.Vector.Shape) != 1 || len(e.Vector.Stride) != 1 || len(e.Output.Shape) != 1 || len(e.Output.Stride) != 1 || e.Matrix.Stride[1] != 1 {
			return fmt.Errorf("operation %d: incompatible GEMV metadata", e.Operation)
		}
		m, n := e.Matrix.Shape[1], e.Matrix.Shape[0]
		normal := e.Trans == "N"
		if normal {
			if e.Vector.Shape[0] != n || e.Output.Shape[0] != m || e.Vector.Stride[0] != 1 || e.Output.Stride[0] != 1 {
				return fmt.Errorf("operation %d: incompatible normal GEMV metadata", e.Operation)
			}
		} else if e.Trans == "T" || e.Trans == "" { // Older captures were transposed only.
			if e.Vector.Shape[0] != m || e.Output.Shape[0] != n || e.Beta != 0 {
				return fmt.Errorf("operation %d: incompatible transposed GEMV metadata", e.Operation)
			}
		} else {
			return fmt.Errorf("unsupported original GEMV transpose %q", e.Trans)
		}
		a, err := read(root, e.Matrix)
		if err != nil {
			return err
		}
		x, err := read(root, e.Vector)
		if err != nil {
			return err
		}
		want, err := read(root, e.Output)
		if err != nil {
			return err
		}
		var y []float32
		if normal {
			var initial []float32
			if e.Beta != 0 {
				if len(e.Initial.Shape) != 1 || len(e.Initial.Stride) != 1 || e.Initial.Shape[0] != m || e.Initial.Stride[0] != 1 {
					return fmt.Errorf("operation %d: missing normal GEMV initial output", e.Operation)
				}
				initial, err = read(root, e.Initial)
				if err != nil {
					return err
				}
			}
			y, err = onnx.MKLGEMVNormal(a, x, initial, m, n, e.Matrix.Stride[0], e.Output.Alignment64%16, e.Alpha, e.Beta)
		} else {
			y, err = onnx.MKLGEMVTranspose(a, x, m, n, e.Matrix.Stride[0], e.Vector.Stride[0], e.Matrix.Alignment64%16, e.Vector.Alignment64%16, e.Alpha)
		}
		if err != nil {
			return err
		}
		c := counts[e.Operation]
		c.calls++
		c.values += len(y)
		for i, value := range y {
			bits := math.Float32bits(want[i*e.Output.Stride[0]])
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
		return fmt.Errorf("no original GEMV results found")
	}
	fmt.Printf("total: %d calls, %d values, %d bit differences; original SHA256 %x\n", total.calls, total.values, total.differences, hash.Sum(nil))
	if skipped != 0 {
		fmt.Printf("not compared: %d GEMV calls without captured values (geometry only)\n", skipped)
	}
	if total.differences != 0 {
		return fmt.Errorf("original GEMV differs")
	}
	return nil
}

func main() {
	root := flag.String("trace", "", "original --torch-gemv-values observation directory")
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
