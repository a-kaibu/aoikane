// Check portable FFT correlation against the native chunk alignment transform.
package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/a-kaibu/aoikane/internal/dsp"
)

type tensor struct {
	ID    int   `json:"id"`
	Shape []int `json:"shape"`
	DType int   `json:"dtype"`
}

type event struct {
	Event  string   `json:"event"`
	Run    int      `json:"run"`
	Inputs []tensor `json:"inputs"`
	Output tensor   `json:"output"`
}

func main() {
	root := flag.String("trace", "private/chunk-correlation-trace", "native trace directory")
	flag.Parse()
	if err := run(*root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(root string) error {
	b, err := os.ReadFile(filepath.Join(root, "events.json"))
	if err != nil {
		return err
	}
	var events []event
	if err := json.Unmarshal(b, &events); err != nil {
		return err
	}
	outputs := map[int]tensor{}
	for _, e := range events {
		if e.Event == "chunk_correlation_result" {
			outputs[e.Run] = e.Output
		}
	}
	load := func(t tensor) ([]float32, error) {
		if t.ID == 0 || t.DType != 6 || len(t.Shape) != 1 {
			return nil, fmt.Errorf("unsupported correlation tensor")
		}
		b, err := os.ReadFile(filepath.Join(root, fmt.Sprintf("model-%d.bin", t.ID)))
		if err != nil {
			return nil, err
		}
		if len(b) != t.Shape[0]*4 {
			return nil, fmt.Errorf("correlation tensor size mismatch")
		}
		out := make([]float32, t.Shape[0])
		for i := range out {
			out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
		}
		return out, nil
	}
	count := 0
	for _, e := range events {
		if e.Event != "chunk_correlation" {
			continue
		}
		if len(e.Inputs) != 2 {
			return fmt.Errorf("unsupported correlation input")
		}
		a, err := load(e.Inputs[0])
		if err != nil {
			return err
		}
		b, err := load(e.Inputs[1])
		if err != nil {
			return err
		}
		expected, err := load(outputs[e.Run])
		if err != nil {
			return err
		}
		actual, err := dsp.CrossCorrelation(a, b)
		if err != nil {
			return err
		}
		if len(actual) != len(expected) {
			return fmt.Errorf("correlation length mismatch")
		}
		var maximum float64
		actualPeak, expectedPeak := 0, 0
		for i, v := range actual {
			d := math.Abs(float64(v) - float64(expected[i]))
			maximum = max(maximum, d)
			if d > 1e-6+1e-5*math.Abs(float64(expected[i])) {
				return fmt.Errorf("correlation differs at index %d", i)
			}
			if v > actual[actualPeak] {
				actualPeak = i
			}
			if expected[i] > expected[expectedPeak] {
				expectedPeak = i
			}
		}
		if actualPeak != expectedPeak {
			return fmt.Errorf("correlation peaks differ: %d vs %d", actualPeak, expectedPeak)
		}
		count++
		if err := json.NewEncoder(os.Stdout).Encode(map[string]any{"run": e.Run, "samples": len(actual), "max_abs_error": maximum, "lag": actualPeak - len(b) + 1, "close": true}); err != nil {
			return err
		}
	}
	if count == 0 {
		return fmt.Errorf("no native chunk correlation calls")
	}
	return nil
}
