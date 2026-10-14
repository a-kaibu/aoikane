// This developer tool compares the Go DSP implementation with captured native
// outputs. Its fixtures and converted window coefficients stay in private/.
package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/a-kaibu/aoikane/internal/dsp"
)

type tensorInfo struct {
	ID     int   `json:"id"`
	Shape  []int `json:"shape"`
	Stride []int `json:"stride"`
	DType  int   `json:"dtype"`
}
type event struct {
	Event  string       `json:"event"`
	Run    int          `json:"run"`
	Inputs []tensorInfo `json:"inputs"`
	Output tensorInfo   `json:"output"`
}
type result struct {
	Run         int     `json:"run"`
	Samples     int     `json:"samples"`
	Seconds     float64 `json:"seconds"`
	MaxAbsError float64 `json:"max_abs_error"`
	MeanError   float64 `json:"mean_abs_error"`
	Close       bool    `json:"close"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	root := filepath.Join("private", "model-trace")
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	var c dsp.Compressor
	var events []event
	for name, target := range map[string]any{"compressor.json": &c, "events.json": &events} {
		b, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return err
		}
		if err = json.Unmarshal(b, target); err != nil {
			return err
		}
	}
	load := func(t tensorInfo) ([]float32, error) {
		if t.DType != 6 || len(t.Shape) != 3 || t.Shape[0] != 1 || t.Shape[1] != 1 || len(t.Stride) != 3 || t.Stride[2] != 1 {
			return nil, fmt.Errorf("unsupported tensor %d", t.ID)
		}
		b, err := os.ReadFile(filepath.Join(root, fmt.Sprintf("model-%d.bin", t.ID)))
		if err != nil {
			return nil, err
		}
		if len(b) != 4*t.Shape[2] {
			return nil, fmt.Errorf("tensor %d size mismatch", t.ID)
		}
		values := make([]float32, t.Shape[2])
		for i := range values {
			values[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
		}
		return values, nil
	}
	outputs := map[int]tensorInfo{}
	for _, e := range events {
		if e.Event == "torch_jit_result" {
			outputs[e.Run] = e.Output
		}
	}
	var results []result
	for _, e := range events {
		if e.Event != "torch_jit_forward" || len(e.Inputs) != 2 {
			continue
		}
		var arrays [3][]float32
		for i, t := range []tensorInfo{e.Inputs[0], e.Inputs[1], outputs[e.Run]} {
			v, err := load(t)
			if err != nil {
				return err
			}
			arrays[i] = v
		}
		start := time.Now()
		actual, err := c.Process(arrays[0], arrays[1])
		if err != nil {
			return err
		}
		r := result{Run: e.Run, Samples: len(actual), Seconds: time.Since(start).Seconds(), Close: true}
		for i, v := range actual {
			d := math.Abs(float64(v) - float64(arrays[2][i]))
			r.MaxAbsError = max(r.MaxAbsError, d)
			r.MeanError += d / float64(len(actual))
			if d > 1e-6+1e-5*math.Abs(float64(arrays[2][i])) {
				r.Close = false
			}
		}
		results = append(results, r)
		b, _ := json.Marshal(r)
		fmt.Println(string(b))
	}
	b, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(root, "go-compressor-results.json"), b, 0600); err != nil {
		return err
	}
	if len(results) == 0 {
		return fmt.Errorf("no native compressor calls found")
	}
	for _, r := range results {
		if !r.Close {
			return fmt.Errorf("Go compressor differs from native output")
		}
	}
	return nil
}
