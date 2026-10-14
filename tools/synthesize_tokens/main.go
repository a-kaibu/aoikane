//go:build (windows || darwin || linux) && (amd64 || arm64)

// Developer CLI for the portable neural pipeline's phoneme input contract.
// The production text CLI will use this pipeline after the frontend is complete.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/a-kaibu/aoikane/internal/engine"
	"github.com/a-kaibu/aoikane/internal/frontend"
	"github.com/a-kaibu/aoikane/internal/synthesis"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	input := flag.String("input", "private/token-inputs.json", "phoneme input JSON")
	kana := flag.String("kana", "", "formatted kana with accent marks (text reading remains upstream)")
	phonetics := flag.String("phonetics", "private/phonetics.json", "phonetic definitions for formatted kana")
	controlsPath := flag.String("controls", "", "voice style and control JSON (required with -kana)")
	dumpInput := flag.String("dump-input", "", "also write generated phoneme inputs for comparison")
	models := flag.String("models", "private/model-trace", "private ONNX model directory")
	name := map[string]string{"windows": "onnxruntime.dll", "linux": "libonnxruntime.so", "darwin": "libonnxruntime.dylib"}[runtime.GOOS]
	library := flag.String("runtime", filepath.Join("private/runtime", name), "CPU ONNX Runtime library")
	output := flag.String("output", "private/portable-speech.wav", "output WAV")
	flag.Parse()
	var inputs []synthesis.Input
	if *kana != "" {
		if *controlsPath == "" {
			return fmt.Errorf("-kana requires -controls")
		}
		var controls synthesis.Controls
		if err := decodeFile(*controlsPath, &controls); err != nil {
			return err
		}
		encoder, err := frontend.OpenMoraEncoder(*phonetics)
		if err != nil {
			return err
		}
		sentence, err := encoder.ParseFormattedKana(*kana)
		if err != nil {
			return err
		}
		sequence, err := encoder.CompilePhones(sentence, 0xef)
		if err != nil {
			return err
		}
		generated, err := synthesis.FromPhones(sequence, controls)
		if err != nil {
			return err
		}
		inputs = []synthesis.Input{generated}
	} else if err := decodeFile(*input, &inputs); err != nil {
		return err
	}
	if len(inputs) == 0 {
		return fmt.Errorf("no phoneme inputs")
	}
	if *dumpInput != "" {
		b, err := json.Marshal(inputs)
		if err != nil {
			return err
		}
		if err := os.WriteFile(*dumpInput, b, 0600); err != nil {
			return err
		}
	}
	e, err := synthesis.Open(*library, *models)
	if err != nil {
		return err
	}
	defer e.Close()
	start := time.Now()
	var pcm []byte
	for _, input := range inputs {
		part, err := e.Render(context.Background(), input)
		if err != nil {
			return err
		}
		pcm = append(pcm, part...)
		if len(pcm) > 256*1024*1024 {
			return fmt.Errorf("audio exceeds 256 MiB")
		}
	}
	wav, err := engine.WAV(pcm, 48000)
	if err != nil {
		return err
	}
	if err := os.WriteFile(*output, wav, 0600); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"sentences": len(inputs), "samples": len(pcm) / 2, "seconds": time.Since(start).Seconds(), "sample_rate": 48000})
}

func decodeFile(path string, value any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	d := json.NewDecoder(io.LimitReader(f, 32*1024*1024))
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return fmt.Errorf("%s must contain one JSON value", path)
	}
	return nil
}
