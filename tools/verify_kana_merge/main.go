package main

import (
	"github.com/a-kaibu/aoikane/internal/frontend"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

type nativeWord struct {
	Surface       []int  `json:"surface"`
	Base          []int  `json:"base"`
	Pronunciation []int  `json:"pronunciation"`
	POS           uint32 `json:"pos"`
	OriginCount   int    `json:"origin_count"`
}

func run() error {
	trace := flag.String("trace", "private/kana-merge-live-trace/events.json", "normal-engine merge events")
	codecPath := flag.String("cp932", "private/cp932.json", "CP932 mapping")
	kanaPath := flag.String("kana", "private/unknown-kana.json", "kana conversion rules")
	accentPath := flag.String("accent", "private/unknown-accent.json", "unknown accent rules")
	flag.Parse()
	codec, err := frontend.OpenCP932Codec(*codecPath)
	if err != nil {
		return err
	}
	kana, err := frontend.OpenUnknownKanaRules(*kanaPath)
	if err != nil {
		return err
	}
	accent, err := frontend.OpenUnknownAccentRules(*accentPath)
	if err != nil {
		return err
	}
	decode := func(values []int) (string, error) {
		b := make([]byte, len(values))
		for i, v := range values {
			if v < 0 || v > 255 {
				return "", fmt.Errorf("invalid trace byte")
			}
			b[i] = byte(v)
		}
		return codec.Decode(b)
	}
	words := func(input []nativeWord) ([]frontend.KanaMergeWord, error) {
		out := make([]frontend.KanaMergeWord, len(input))
		for i, w := range input {
			surface, err := decode(w.Surface)
			if err != nil {
				return nil, err
			}
			base, err := decode(w.Base)
			if err != nil {
				return nil, err
			}
			pron, err := decode(w.Pronunciation)
			if err != nil {
				return nil, err
			}
			out[i] = frontend.KanaMergeWord{Surface: surface, Base: base, POS: w.POS, Pronunciation: pron}
		}
		return out, nil
	}
	b, err := os.ReadFile(*trace)
	if err != nil {
		return err
	}
	var events []json.RawMessage
	if err := json.Unmarshal(b, &events); err != nil {
		return err
	}
	count, changed, groups, origins, modelResults := 0, 0, 0, 0, 0
	for _, raw := range events {
		var header struct {
			Event string `json:"event"`
		}
		if err := json.Unmarshal(raw, &header); err != nil {
			return err
		}
		if header.Event == "frontend_error" || header.Event == "forward_error" {
			return fmt.Errorf("failed engine trace")
		}
		if header.Event == "torch_jit_result" {
			modelResults++
		}
		if header.Event != "frontend_kana_merge" && header.Event != "frontend_kana_group" {
			continue
		}
		var event struct {
			Before []nativeWord       `json:"before"`
			After  []nativeWord       `json:"after"`
			Runs   []frontend.KanaRun `json:"runs"`
		}
		if err := json.Unmarshal(raw, &event); err != nil {
			return err
		}
		input, err := words(event.Before)
		if err != nil {
			return err
		}
		want, err := words(event.After)
		if err != nil {
			return err
		}
		var got []frontend.KanaMergeWord
		if header.Event == "frontend_kana_group" {
			group := make([]frontend.KanaGroupWord, len(input))
			for i, w := range input {
				group[i] = frontend.KanaGroupWord{KanaMergeWord: w, OriginCount: event.Before[i].OriginCount}
				if group[i].OriginCount > 0 {
					origins++
				}
			}
			runs, err := frontend.SelectKanaRuns(group)
			if err != nil {
				return err
			}
			if len(runs) != len(event.Runs) {
				return fmt.Errorf("group %d: got %v runs want %v", groups, runs, event.Runs)
			}
			for i := range runs {
				if runs[i] != event.Runs[i] {
					return fmt.Errorf("group %d run %d: got %+v want %+v", groups, i, runs[i], event.Runs[i])
				}
			}
			got, err = frontend.MergeKanaGroup(group, codec, kana, accent)
			groups++
		} else {
			got, err = frontend.MergeKanaRun(input, codec, kana, accent)
		}
		if err != nil {
			return err
		}
		if len(got) != len(want) {
			return fmt.Errorf("merge %d: got %d words want %d", count, len(got), len(want))
		}
		for i, w := range want {
			if got[i].Surface != w.Surface || got[i].Base != w.Base || got[i].POS != w.POS || got[i].Pronunciation != w.Pronunciation {
				return fmt.Errorf("merge %d word %d got %+v want %+v", count, i, got[i], w)
			}
		}
		if len(got) != len(input) {
			changed++
		}
		count++
	}
	if count == 0 {
		return fmt.Errorf("no native kana merge events")
	}
	if modelResults == 0 {
		return fmt.Errorf("trace did not reach model inference")
	}
	fmt.Printf("Verified %d native merge events (%d reduced word count), including %d full groups (%d origin-bearing words)\n", count, changed, groups, origins)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
