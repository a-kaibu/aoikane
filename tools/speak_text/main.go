//go:build (windows || darwin || linux) && (amd64 || arm64)

// Developer CLI: Japanese text to speech with the portable pipeline only
// (reconstructed standard-voice frontend, ONNX acoustic model and vocoder).
// Assets are the decoded private files; nothing here calls the original engine.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/a-kaibu/aoikane/internal/cabocha"

	"github.com/a-kaibu/aoikane/internal/engine"
	"github.com/a-kaibu/aoikane/internal/frontend"
	"github.com/a-kaibu/aoikane/internal/numeral"
	"github.com/a-kaibu/aoikane/internal/phrase"
	"github.com/a-kaibu/aoikane/internal/synthesis"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	private := flag.String("private", "private", "directory with the decoded private assets")
	models := flag.String("models", "private/model-trace", "private ONNX model directory (featnet, vocoder)")
	controlsPath := flag.String("controls", "private/kana-controls.json", "voice style and control JSON")
	phonetics := flag.String("phonetics", "private/phonetics.json", "phonetic definitions")
	name := map[string]string{"windows": "onnxruntime.dll", "linux": "libonnxruntime.so", "darwin": "libonnxruntime.dylib"}[runtime.GOOS]
	library := flag.String("runtime", filepath.Join("private/runtime", name), "CPU ONNX Runtime library")
	output := flag.String("output", "speech.wav", "output WAV")
	dumpInput := flag.String("dump-input", "", "save generated phoneme model inputs for native comparison")
	file := flag.String("file", "", "read the text from this UTF-8 file")
	flag.Parse()
	text := ""
	switch {
	case *file != "":
		b, err := os.ReadFile(*file)
		if err != nil {
			return err
		}
		text = string(b)
	case flag.NArg() > 0:
		text = flag.Arg(0)
	default:
		return fmt.Errorf("provide the text or -file")
	}
	fe, closeFrontend, err := openFrontend(*private, *library)
	if err != nil {
		return err
	}
	defer closeFrontend()
	kana, err := fe.KanaText(text)
	if err != nil {
		return err
	}
	var controls synthesis.Controls
	b, err := os.ReadFile(*controlsPath)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, &controls); err != nil {
		return err
	}
	encoder, err := frontend.OpenMoraEncoder(*phonetics)
	if err != nil {
		return err
	}
	e, err := synthesis.Open(*library, *models)
	if err != nil {
		return err
	}
	defer e.Close()
	var pcm []byte
	var inputs []synthesis.Input
	for _, k := range kana {
		fmt.Fprintln(os.Stderr, k)
		sentence, err := encoder.ParseFormattedKana(k)
		if err != nil {
			return err
		}
		sequence, err := encoder.CompilePhones(sentence, 0xef)
		if err != nil {
			return err
		}
		input, err := synthesis.FromPhones(sequence, controls)
		if err != nil {
			return err
		}
		inputs = append(inputs, input)
		part, err := e.Render(context.Background(), input)
		if err != nil {
			return err
		}
		pcm = append(pcm, part...)
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
	wav, err := engine.WAV(pcm, 48000)
	if err != nil {
		return err
	}
	return os.WriteFile(*output, wav, 0o600)
}

func openFrontend(private, library string) (*phrase.Frontend, func(), error) {
	analyzer, err := frontend.OpenAnalyzer(filepath.Join(private, "frontend-data"))
	if err != nil {
		return nil, nil, err
	}
	morphology, err := frontend.OpenMorphology(filepath.Join(private, "morphology.json"))
	if err != nil {
		return nil, nil, err
	}
	rf, err := os.Open(filepath.Join(private, "reading-rules-utf8.txt"))
	if err != nil {
		return nil, nil, err
	}
	defer rf.Close()
	reading, err := frontend.ParseReadingRules(rf)
	if err != nil {
		return nil, nil, err
	}
	reading.SkipPhraseScopes = true
	codec, err := frontend.OpenCP932Codec(filepath.Join(private, "cp932.json"))
	if err != nil {
		return nil, nil, err
	}
	bf, err := os.Open(filepath.Join(private, "bound-rules-standard.txt"))
	if err != nil {
		return nil, nil, err
	}
	defer bf.Close()
	bound, err := phrase.ParseBoundRules(bf)
	if err != nil {
		return nil, nil, err
	}
	model, err := phrase.OpenBoundModel(library, "assets/boundmodel", codec)
	if err != nil {
		return nil, nil, err
	}
	nf, err := os.Open("assets/rules/bancharcx-standard.txt")
	if err != nil {
		return nil, nil, err
	}
	defer nf.Close()
	numerals, err := numeral.ParseRules(nf)
	if err != nil {
		return nil, nil, err
	}
	uk, err := frontend.OpenUnknownKanaRules("assets/rules/unknown-kana.json")
	if err != nil {
		model.Close()
		return nil, nil, err
	}
	ua, err := frontend.OpenUnknownAccentRules("assets/rules/unknown-accent.json")
	if err != nil {
		model.Close()
		return nil, nil, err
	}
	english, err := frontend.OpenEnglishReader("assets/rules", codec, ua)
	if err != nil {
		model.Close()
		return nil, nil, err
	}
	chunker, err := cabocha.LoadCRF("assets/cabocha/standard-chunk.ipa.model")
	if err != nil {
		model.Close()
		return nil, nil, err
	}
	return &phrase.Frontend{Analyzer: analyzer, Morphology: morphology, Reading: reading, Codec: codec, Bound: bound, Predictor: model, Numerals: numerals, UnknownKana: uk, UnknownAccent: ua, English: english, Chunker: chunker}, model.Close, nil
}
