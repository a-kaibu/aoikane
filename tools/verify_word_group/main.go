package main

import (
	"github.com/a-kaibu/aoikane/internal/frontend"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func run() error {
	root := flag.String("rules", "private", "directory of exported rules")
	trace := flag.String("trace", "private/word-group-live-trace/events.json", "normal word-group trace")
	flag.Parse()
	path := func(name string) string { return filepath.Join(*root, name+".json") }
	codec, err := frontend.OpenCP932Codec(path("cp932"))
	if err != nil {
		return err
	}
	kana, err := frontend.OpenUnknownKanaRules(path("unknown-kana"))
	if err != nil {
		return err
	}
	accent, err := frontend.OpenUnknownAccentRules(path("unknown-accent"))
	if err != nil {
		return err
	}
	spelling, err := frontend.OpenEnglishSpellingRules(path("english-spelling"))
	if err != nil {
		return err
	}
	transliteration, err := frontend.OpenEnglishTransliterationRules(path("english-transliteration"))
	if err != nil {
		return err
	}
	segments, err := frontend.OpenEnglishSegmentRules(path("english-segments"))
	if err != nil {
		return err
	}
	trees, err := frontend.OpenEnglishDecisionTrees(path("english-decisions"))
	if err != nil {
		return err
	}
	englishKana, err := frontend.OpenEnglishKanaRules(path("english-kana"))
	if err != nil {
		return err
	}
	fallback := &frontend.EnglishFallbackRules{Segments: segments, Trees: trees, Kana: englishKana, Codec: codec}
	read := func(s string) (frontend.UnknownKanaReading, error) {
		return frontend.CompileUnknownEnglish(s, spelling, transliteration, accent, fallback)
	}
	type nativeWord struct {
		Surface       []byte `json:"surface"`
		Base          []byte `json:"base"`
		Pronunciation []byte `json:"pronunciation"`
		POS           uint32 `json:"pos"`
		OriginCount   int    `json:"origin_count"`
	}
	decode := func(input []nativeWord) ([]frontend.KanaGroupWord, error) {
		words := make([]frontend.KanaGroupWord, len(input))
		for i, w := range input {
			surface, err := codec.Decode(w.Surface)
			if err != nil {
				return nil, err
			}
			base, err := codec.Decode(w.Base)
			if err != nil {
				return nil, err
			}
			pron, err := codec.Decode(w.Pronunciation)
			if err != nil {
				return nil, err
			}
			words[i] = frontend.KanaGroupWord{KanaMergeWord: frontend.KanaMergeWord{Surface: surface, Base: base, Pronunciation: pron, POS: w.POS}, OriginCount: w.OriginCount}
		}
		return words, nil
	}
	b, err := os.ReadFile(*trace)
	if err != nil {
		return err
	}
	var events []json.RawMessage
	if err := json.Unmarshal(b, &events); err != nil {
		return err
	}
	groups, models, changed := 0, 0, 0
	for _, raw := range events {
		var header struct {
			Event string `json:"event"`
		}
		if err := json.Unmarshal(raw, &header); err != nil {
			return err
		}
		switch header.Event {
		case "frontend_error", "forward_error":
			return fmt.Errorf("failed engine trace")
		case "torch_jit_result":
			models++
		case "frontend_word_group":
			var e struct {
				Before []nativeWord `json:"before"`
				After  []nativeWord `json:"after"`
			}
			if err := json.Unmarshal(raw, &e); err != nil {
				return err
			}
			before, err := decode(e.Before)
			if err != nil {
				return err
			}
			after, err := decode(e.After)
			if err != nil {
				return err
			}
			got, err := frontend.MergeKanaEnglishGroup(before, codec, kana, accent, read)
			if err != nil {
				return err
			}
			if len(got) != len(after) {
				return fmt.Errorf("group %d length got %d want %d", groups, len(got), len(after))
			}
			for i, w := range after {
				if got[i].Surface != w.Surface || got[i].Base != w.Base || got[i].Pronunciation != w.Pronunciation || got[i].POS != w.POS {
					return fmt.Errorf("group %d word %d differs", groups, i)
				}
			}
			if len(before) != len(after) {
				changed++
			}
			groups++
		}
	}
	if models == 0 || groups == 0 {
		return fmt.Errorf("incomplete word-group trace")
	}
	fmt.Printf("Verified %d combined kana/English word groups (%d changed lengths)\n", groups, changed)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
