//go:build (windows || darwin || linux) && (amd64 || arm64)

// Developer CLI: text to the engine's formatted kana with accent marks, using
// the reconstructed standard-voice frontend and private decoded assets.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/a-kaibu/aoikane/internal/cabocha"

	"github.com/a-kaibu/aoikane/internal/frontend"
	"github.com/a-kaibu/aoikane/internal/numeral"
	"github.com/a-kaibu/aoikane/internal/phrase"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	private := flag.String("private", "private", "directory with the decoded private assets")
	name := map[string]string{"windows": "onnxruntime.dll", "linux": "libonnxruntime.so", "darwin": "libonnxruntime.dylib"}[runtime.GOOS]
	library := flag.String("runtime", filepath.Join("private/runtime", name), "CPU ONNX Runtime library")
	flag.Parse()
	if flag.NArg() == 0 {
		return fmt.Errorf("provide input text")
	}
	analyzer, err := frontend.OpenAnalyzer(filepath.Join(*private, "frontend-data"))
	if err != nil {
		return err
	}
	morphology, err := frontend.OpenMorphology(filepath.Join(*private, "morphology.json"))
	if err != nil {
		return err
	}
	rf, err := os.Open(filepath.Join(*private, "reading-rules-utf8.txt"))
	if err != nil {
		return err
	}
	defer rf.Close()
	reading, err := frontend.ParseReadingRules(rf)
	if err != nil {
		return err
	}
	reading.SkipPhraseScopes = true
	codec, err := frontend.OpenCP932Codec(filepath.Join(*private, "cp932.json"))
	if err != nil {
		return err
	}
	bf, err := os.Open(filepath.Join(*private, "bound-rules-standard.txt"))
	if err != nil {
		return err
	}
	defer bf.Close()
	bound, err := phrase.ParseBoundRules(bf)
	if err != nil {
		return err
	}
	model, err := phrase.OpenBoundModel(*library, "assets/boundmodel", codec)
	if err != nil {
		return err
	}
	defer model.Close()
	nf, err := os.Open("assets/rules/bancharcx-standard.txt")
	if err != nil {
		return err
	}
	defer nf.Close()
	numerals, err := numeral.ParseRules(nf)
	if err != nil {
		return err
	}
	uk, err := frontend.OpenUnknownKanaRules("assets/rules/unknown-kana.json")
	if err != nil {
		return err
	}
	ua, err := frontend.OpenUnknownAccentRules("assets/rules/unknown-accent.json")
	if err != nil {
		return err
	}
	english, err := frontend.OpenEnglishReader("assets/rules", codec, ua)
	if err != nil {
		return err
	}
	chunker, err := cabocha.LoadCRF("assets/cabocha/standard-chunk.ipa.model")
	if err != nil {
		return err
	}
	fe := &phrase.Frontend{Analyzer: analyzer, Morphology: morphology, Reading: reading, Codec: codec, Bound: bound, Predictor: model, Numerals: numerals, UnknownKana: uk, UnknownAccent: ua, English: english, Chunker: chunker}
	for _, text := range flag.Args() {
		sentences, err := fe.KanaText(text)
		if err != nil {
			return err
		}
		for _, kana := range sentences {
			fmt.Println(kana)
		}
	}
	return nil
}
