//go:build (windows || darwin || linux) && (amd64 || arm64)

package main

import (
	"github.com/a-kaibu/aoikane/internal/difftest"
	"github.com/a-kaibu/aoikane/internal/frontend"
	"github.com/a-kaibu/aoikane/internal/portable"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"slices"
)

type capture struct {
	DLLHash         string `json:"dll_sha256"`
	Voice           string
	Complete        bool
	TimedOut        bool `json:"timed_out"`
	Errors          []json.RawMessage
	Detach          string `json:"detach_reason"`
	Crash           json.RawMessage
	CorpusSHA       string `json:"corpus_sha256"`
	NativeReportSHA string `json:"native_report_sha256"`
	Cases           []struct {
		Index    int
		Text     string
		Status   int
		Readings []string
		CP932    []string `json:"cp932_hex"`
		WAVBytes int      `json:"wav_bytes"`
		WAVSHA   string   `json:"wav_sha256"`
	}
}

func digestValid(s string) bool {
	b, err := hex.DecodeString(s)
	return err == nil && len(b) == sha256.Size
}

func (r capture) validate() error {
	if !r.Complete || r.TimedOut || len(r.Errors) != 0 || r.Detach != "process-terminated" || !bytes.Equal(bytes.TrimSpace(r.Crash), []byte("null")) || !digestValid(r.CorpusSHA) || !digestValid(r.NativeReportSHA) || r.DLLHash != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(r.Cases) < 1 || len(r.Cases) > 512 || (r.Voice != portable.VoiceAoi && r.Voice != portable.VoiceAkane) {
		return fmt.Errorf("original capture provenance/integrity differs")
	}
	for i, c := range r.Cases {
		if c.Index != i || c.Status != 0 || c.Text == "" || len(c.Readings) == 0 || len(c.CP932) != len(c.Readings) || c.WAVBytes < 44 || !digestValid(c.WAVSHA) {
			return fmt.Errorf("original case %d is incomplete or out of order", i)
		}
		for j, s := range c.CP932 {
			b, err := hex.DecodeString(s)
			if err != nil || len(b) == 0 || c.Readings[j] == "" {
				return fmt.Errorf("original case %d has no valid formatter output", i)
			}
		}
	}
	return nil
}

func run() error {
	generate := flag.Bool("generate", false, "write deterministic input corpus instead of comparing")
	numerals := flag.Bool("numerals", false, "generate numeral/unit/counter stress inputs; existing mixed generator remains unchanged")
	seed := flag.Uint64("seed", 5, "random seed")
	count := flag.Int("count", 128, "input count")
	units := flag.Int("max-units", 12, "maximum generator units")
	output := flag.String("output", "", "corpus or comparison report path")
	reference := flag.String("reference", "", "original frontend capture")
	flag.Parse()
	if *output == "" {
		return fmt.Errorf("output required")
	}
	if *generate {
		if *count < 1 || *count > 512 || *units < 1 || *units > 128 {
			return fmt.Errorf("invalid corpus bounds")
		}
		corpus := map[string]any{"seed": *seed, "count": *count, "generator_version": difftest.GeneratorVersion, "max_units": *units, "texts": difftest.Texts(*seed, *count, *units)}
		if *numerals {
			corpus["generator_kind"] = "numeral"
			corpus["texts"] = difftest.NumeralTexts(*seed, *count, *units)
		}
		raw, err := json.MarshalIndent(corpus, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(*output, append(raw, '\n'), 0600)
	}
	var ref capture
	raw, err := os.ReadFile(*reference)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(raw, &ref); err != nil {
		return err
	}
	if err = ref.validate(); err != nil {
		return err
	}
	sourceSHA := fmt.Sprintf("%x", sha256.Sum256(raw))
	codec, err := frontend.OpenCP932Codec("assets/data/cp932.json")
	if err != nil {
		return err
	}
	for i, c := range ref.Cases {
		for j, h := range c.CP932 {
			b, _ := hex.DecodeString(h) // validated above
			s, err := codec.Decode(b)
			if err != nil || s != c.Readings[j] {
				return fmt.Errorf("original case %d reading does not match captured CP932 bytes", i)
			}
		}
	}
	e, err := portable.Open(portable.Options{AssetsDir: "assets"})
	if err != nil {
		return err
	}
	defer e.Close()
	type row struct {
		Index        int    `json:"index"`
		Text         string `json:"text"`
		Original, Go []string
		Equal        bool   `json:"equal"`
		Error        string `json:"error,omitempty"`
	}
	rows := []row{}
	different := 0
	for i, c := range ref.Cases {
		got, err := e.ReadingVoice(c.Text, ref.Voice)
		r := row{Index: i, Text: c.Text, Original: c.Readings, Go: got, Equal: err == nil && slices.Equal(got, c.Readings)}
		if err != nil {
			r.Error = err.Error()
		}
		if !r.Equal {
			different++
			fmt.Printf("different case=%d text=%q original=%q Go=%q err=%v\n", i, c.Text, c.Readings, got, err)
		}
		rows = append(rows, r)
	}
	result := map[string]any{"voice": ref.Voice, "cases": rows, "count": len(rows), "different": different, "source_reference": *reference, "source_reference_sha256": sourceSHA, "corpus_sha256": ref.CorpusSHA, "native_report_sha256": ref.NativeReportSHA}
	raw, err = json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(*output, append(raw, '\n'), 0600); err != nil {
		return err
	}
	fmt.Printf("%s: %d original frontend cases, %d differences\n", ref.Voice, len(rows), different)
	if different != 0 {
		return fmt.Errorf("pure Go reading differs from original")
	}
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
