package main

import (
	"github.com/a-kaibu/aoikane/internal/frontend"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"reflect"
)

func run() error {
	tables := flag.String("tables", "private/english-kana.json", "ordered kana rules")
	reference := flag.String("reference", "private/english-kana-reference.json", "native conversions")
	trace := flag.String("trace", "", "normal synthesis trace")
	segments := flag.String("segments", "private/english-segments.json", "segment rules")
	decisions := flag.String("decisions", "private/english-decisions.json", "decision trees")
	cp932 := flag.String("cp932", "private/cp932.json", "CP932 mapping")
	appendReference := flag.String("append-reference", "private/english-append-reference.json", "native append cases")
	fallbackReference := flag.String("fallback-reference", "private/english-fallback-reference.json", "complete native fallback cases")
	flag.Parse()
	rules, err := frontend.OpenEnglishKanaRules(*tables)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(*reference)
	if err != nil {
		return err
	}
	var ref struct {
		Source string `json:"source_sha256"`
		Cases  []struct {
			Reading string `json:"reading"`
			Kana    string `json:"kana"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(b, &ref); err != nil {
		return err
	}
	if ref.Source != rules.Source || len(ref.Cases) == 0 {
		return fmt.Errorf("invalid kana reference")
	}
	for i, c := range ref.Cases {
		got, err := rules.Convert(c.Reading)
		if err != nil {
			return err
		}
		if got != c.Kana {
			return fmt.Errorf("case %d %q got %q want %q", i, c.Reading, got, c.Kana)
		}
	}
	fmt.Printf("Verified %d native phonetic-to-kana conversions\n", len(ref.Cases))
	codec, err := frontend.OpenCP932Codec(*cp932)
	if err != nil {
		return err
	}
	type appendCase struct {
		Reading   string `json:"reading"`
		PrefixHex string `json:"prefix_hex"`
		ResultHex string `json:"result_hex"`
		Capacity  int    `json:"capacity"`
		Length    int    `json:"length"`
	}
	checkAppend := func(c appendCase) ([]byte, error) {
		prefix, err := hex.DecodeString(c.PrefixHex)
		if err != nil {
			return nil, err
		}
		want, err := hex.DecodeString(c.ResultHex)
		if err != nil {
			return nil, err
		}
		got, err := rules.AppendCP932(prefix, c.Reading, c.Capacity, codec)
		if err != nil {
			return nil, err
		}
		if len(got) != c.Length || !bytes.Equal(got, want) {
			return nil, fmt.Errorf("English append mismatch for %q limit %d", c.Reading, c.Capacity)
		}
		return got, nil
	}
	b, err = os.ReadFile(*appendReference)
	if err != nil {
		return err
	}
	var appends struct {
		Source string       `json:"source_sha256"`
		Cases  []appendCase `json:"cases"`
	}
	if err := json.Unmarshal(b, &appends); err != nil {
		return err
	}
	if appends.Source != rules.Source || len(appends.Cases) == 0 {
		return fmt.Errorf("invalid append reference")
	}
	for i, c := range appends.Cases {
		if _, err := checkAppend(c); err != nil {
			return fmt.Errorf("append case %d: %w", i, err)
		}
	}
	fmt.Printf("Verified %d native kana append cases\n", len(appends.Cases))
	segmentRules, err := frontend.OpenEnglishSegmentRules(*segments)
	if err != nil {
		return err
	}
	trees, err := frontend.OpenEnglishDecisionTrees(*decisions)
	if err != nil {
		return err
	}
	b, err = os.ReadFile(*fallbackReference)
	if err != nil {
		return err
	}
	var fallback struct {
		Source string `json:"source_sha256"`
		Cases  []struct {
			Text string `json:"text"`
			Kana string `json:"kana"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(b, &fallback); err != nil {
		return err
	}
	if fallback.Source != rules.Source || len(fallback.Cases) == 0 {
		return fmt.Errorf("invalid fallback reference")
	}
	for i, c := range fallback.Cases {
		got, err := segmentRules.ResolveEnglishFallback(c.Text, trees, rules, codec)
		if err != nil {
			return fmt.Errorf("fallback case %d: %w", i, err)
		}
		if got != c.Kana {
			return fmt.Errorf("fallback case %d %q got %q want %q", i, c.Text, got, c.Kana)
		}
	}
	fmt.Printf("Verified %d complete native English fallback cases\n", len(fallback.Cases))
	if *trace == "" {
		return nil
	}
	b, err = os.ReadFile(*trace)
	if err != nil {
		return err
	}
	var events []json.RawMessage
	if err := json.Unmarshal(b, &events); err != nil {
		return err
	}
	var readings []string
	position, chunks, count, models := 0, 0, 0, 0
	appendCount := 0
	fallbackCount := 0
	var prefix []byte
	var chunkKana string
	for _, raw := range events {
		var e struct {
			Event    string   `json:"event"`
			Text     string   `json:"text"`
			Readings []string `json:"readings"`
			Reading  string   `json:"reading"`
			KanaHex  string   `json:"kana_hex"`
		}
		if err := json.Unmarshal(raw, &e); err != nil {
			return err
		}
		switch e.Event {
		case "frontend_error", "forward_error":
			return fmt.Errorf("failed native trace")
		case "torch_jit_result":
			models++
		case "frontend_english_initial_candidates":
			if position != len(readings) {
				return fmt.Errorf("unfinished kana chunk")
			}
			chunk, err := segmentRules.ResolvePronunciationChunk(e.Text, trees)
			if err != nil {
				return err
			}
			readings, position = chunk.Readings, 0
			prefix = nil
			chunkKana, err = segmentRules.ResolveEnglishFallback(e.Text, trees, rules, codec)
			if err != nil {
				return err
			}
			chunks++
		case "frontend_english_final_candidates":
			if !reflect.DeepEqual(readings, e.Readings) {
				return fmt.Errorf("Go phonetic chunk mismatch")
			}
		case "frontend_english_kana":
			if position >= len(readings) || readings[position] != e.Reading {
				return fmt.Errorf("unexpected native kana input %q", e.Reading)
			}
			got, err := rules.Convert(readings[position])
			if err != nil {
				return err
			}
			bytes, err := hex.DecodeString(e.KanaHex)
			if err != nil {
				return err
			}
			want, err := codec.Decode(bytes)
			if err != nil {
				return err
			}
			if got != want {
				return fmt.Errorf("normal kana got %q want %q", got, want)
			}
			position++
			count++
		case "frontend_english_append":
			var c appendCase
			if err := json.Unmarshal(raw, &c); err != nil {
				return err
			}
			if position < 1 || readings[position-1] != c.Reading || hex.EncodeToString(prefix) != c.PrefixHex {
				return fmt.Errorf("native append input differs from Go chain")
			}
			got, err := checkAppend(c)
			if err != nil {
				return err
			}
			prefix = got
			if position == len(readings) {
				gotKana, err := codec.Decode(prefix)
				if err != nil {
					return err
				}
				if gotKana != chunkKana {
					return fmt.Errorf("connected kana chunk mismatch")
				}
			}
			appendCount++
		case "frontend_english_fallback":
			var f struct {
				Input         string `json:"input"`
				Pronunciation []byte `json:"pronunciation"`
			}
			if err := json.Unmarshal(raw, &f); err != nil {
				return err
			}
			got, err := segmentRules.ResolveEnglishFallback(f.Input, trees, rules, codec)
			if err != nil {
				return err
			}
			want, err := codec.Decode(f.Pronunciation)
			if err != nil {
				return err
			}
			if got != want {
				return fmt.Errorf("normal full fallback %q got %q want %q", f.Input, got, want)
			}
			fallbackCount++
		}
	}
	if models == 0 || count == 0 || chunks == 0 || position != len(readings) {
		return fmt.Errorf("incomplete kana trace")
	}
	fmt.Printf("Verified %d normal kana conversions through %d Go English pronunciation chunks\n", count, chunks)
	if appendCount > 0 && appendCount != count {
		return fmt.Errorf("incomplete normal append chain")
	}
	fmt.Printf("Verified %d normal kana appends\n", appendCount)
	if fallbackCount != chunks {
		return fmt.Errorf("missing normal fallback outputs")
	}
	fmt.Printf("Verified %d complete normal English fallback outputs\n", fallbackCount)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
