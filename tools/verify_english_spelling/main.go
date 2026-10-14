package main

import (
	"github.com/a-kaibu/aoikane/internal/frontend"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"reflect"
)

type nativeWord struct {
	Surface        []byte `json:"surface"`
	Base           []byte `json:"base"`
	Pronunciation  []byte `json:"pronunciation"`
	POS            uint32 `json:"pos"`
	OriginCount    int    `json:"origin_count"`
	BoundaryBefore bool   `json:"boundary_before"`
}

func run() error {
	tables := flag.String("tables", "private/english-spelling.json", "spelling rules")
	reference := flag.String("reference", "private/english-spelling-reference.json", "native references")
	accentPath := flag.String("accent", "private/unknown-accent.json", "mora counting rules")
	trace := flag.String("trace", "", "optional normal-engine events")
	codecPath := flag.String("cp932", "private/cp932.json", "CP932 mapping")
	transliterationPath := flag.String("transliteration", "private/english-transliteration.json", "roman-letter conversion rules")
	transliterationReference := flag.String("transliteration-reference", "private/english-transliteration-reference.json", "roman-letter native references")
	segmentsPath := flag.String("segments", "private/english-segments.json", "fallback segments")
	treesPath := flag.String("decisions", "private/english-decisions.json", "fallback trees")
	kanaPath := flag.String("kana", "private/english-kana.json", "fallback kana rules")
	flag.Parse()
	rules, err := frontend.OpenEnglishSpellingRules(*tables)
	if err != nil {
		return err
	}
	accent, err := frontend.OpenUnknownAccentRules(*accentPath)
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
			Text          string `json:"text"`
			Pronunciation string `json:"pronunciation"`
			Down          int    `json:"down"`
			Moras         int    `json:"moras"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(b, &ref); err != nil {
		return err
	}
	if ref.Source != rules.Source || len(ref.Cases) == 0 {
		return fmt.Errorf("invalid spelling reference")
	}
	for i, c := range ref.Cases {
		got, err := rules.Spell(c.Text, accent)
		if err != nil {
			return err
		}
		if got.Pronunciation != c.Pronunciation || got.Accent.Down != c.Down || got.Accent.Moras != c.Moras {
			return fmt.Errorf("case %d %q got %+v want %q %d/%d", i, c.Text, got, c.Pronunciation, c.Down, c.Moras)
		}
	}
	fmt.Printf("Verified %d native alphabet spellings and accents\n", len(ref.Cases))
	transliteration, err := frontend.OpenEnglishTransliterationRules(*transliterationPath)
	if err != nil {
		return err
	}
	b, err = os.ReadFile(*transliterationReference)
	if err != nil {
		return err
	}
	var transRef struct {
		Source string `json:"source_sha256"`
		Cases  []struct {
			Text          string `json:"text"`
			Pronunciation string `json:"pronunciation"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(b, &transRef); err != nil {
		return err
	}
	if transRef.Source != transliteration.Source || len(transRef.Cases) == 0 {
		return fmt.Errorf("invalid transliteration reference")
	}
	for i, c := range transRef.Cases {
		got, err := transliteration.Transliterate(c.Text)
		if err != nil {
			return err
		}
		if got != c.Pronunciation {
			return fmt.Errorf("transliteration %d %q got %q want %q", i, c.Text, got, c.Pronunciation)
		}
	}
	fmt.Printf("Verified %d native roman-letter conversions\n", len(transRef.Cases))
	if *trace == "" {
		return nil
	}
	codec, err := frontend.OpenCP932Codec(*codecPath)
	if err != nil {
		return err
	}
	segments, err := frontend.OpenEnglishSegmentRules(*segmentsPath)
	if err != nil {
		return err
	}
	trees, err := frontend.OpenEnglishDecisionTrees(*treesPath)
	if err != nil {
		return err
	}
	kana, err := frontend.OpenEnglishKanaRules(*kanaPath)
	if err != nil {
		return err
	}
	fallback := &frontend.EnglishFallbackRules{Segments: segments, Trees: trees, Kana: kana, Codec: codec}
	decodeWords := func(words []nativeWord) ([]frontend.KanaMergeWord, error) {
		out := make([]frontend.KanaMergeWord, len(words))
		for i, w := range words {
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
			out[i] = frontend.KanaMergeWord{Surface: surface, Base: base, Pronunciation: pron, POS: w.POS}
		}
		return out, nil
	}
	b, err = os.ReadFile(*trace)
	if err != nil {
		return err
	}
	var events []json.RawMessage
	if err := json.Unmarshal(b, &events); err != nil {
		return err
	}
	spellings, merges, modelResults, conversions, accents := 0, 0, 0, 0, 0
	unknowns := 0
	groups, originWords, boundaryWords := 0, 0, 0
	for _, raw := range events {
		var e struct {
			Event         string       `json:"event"`
			Input         string       `json:"input"`
			Pronunciation []byte       `json:"pronunciation"`
			Down          int          `json:"down"`
			Before        []nativeWord `json:"before"`
			After         []nativeWord `json:"after"`
		}
		// Other events can have a different input type; decode the header first.
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
			modelResults++
		case "frontend_english_group":
			var c struct {
				Before []nativeWord       `json:"before"`
				After  []nativeWord       `json:"after"`
				Runs   []frontend.KanaRun `json:"runs"`
				End    int                `json:"end"`
			}
			if err := json.Unmarshal(raw, &c); err != nil {
				return err
			}
			before, err := decodeWords(c.Before)
			if err != nil {
				return err
			}
			after, err := decodeWords(c.After)
			if err != nil {
				return err
			}
			words := make([]frontend.EnglishGroupWord, len(before))
			for i, w := range before {
				words[i] = frontend.EnglishGroupWord{KanaGroupWord: frontend.KanaGroupWord{KanaMergeWord: w, OriginCount: c.Before[i].OriginCount}, BoundaryBefore: c.Before[i].BoundaryBefore}
				if c.Before[i].OriginCount > 0 {
					originWords++
				}
				if c.Before[i].BoundaryBefore {
					boundaryWords++
				}
			}
			selection, err := frontend.SelectEnglishRuns(words)
			if err != nil {
				return err
			}
			if selection.End != c.End || !reflect.DeepEqual(selection.Runs, c.Runs) {
				return fmt.Errorf("English group %d selection got %+v want %v/%d", groups, selection, c.Runs, c.End)
			}
			got, end, err := frontend.MergeEnglishGroupScan(words, codec, func(s string) (frontend.UnknownKanaReading, error) {
				return frontend.CompileUnknownEnglish(s, rules, transliteration, accent, fallback)
			})
			if err != nil {
				return err
			}
			if end != c.End || len(got) != len(after) {
				return fmt.Errorf("English group result length mismatch")
			}
			for i, w := range after {
				if got[i].Surface != w.Surface || got[i].Base != w.Base || got[i].POS != w.POS || got[i].Pronunciation != w.Pronunciation {
					return fmt.Errorf("English group %d word %d differs", groups, i)
				}
			}
			groups++
		case "frontend_unknown_english":
			var c struct {
				Before []nativeWord `json:"before"`
				After  []nativeWord `json:"after"`
				Down   *int         `json:"down"`
				Moras  *int         `json:"moras"`
			}
			if err := json.Unmarshal(raw, &c); err != nil {
				return err
			}
			before, err := decodeWords(c.Before)
			if err != nil {
				return err
			}
			after, err := decodeWords(c.After)
			if err != nil {
				return err
			}
			if len(before) != 1 || len(after) != 1 {
				return fmt.Errorf("invalid unknown English word trace")
			}
			got, err := frontend.CompileUnknownEnglish(before[0].Surface, rules, transliteration, accent, fallback)
			if err != nil {
				return err
			}
			if got.Pronunciation != after[0].Pronunciation || after[0].POS != 0x01010400 || before[0].Surface != after[0].Surface || before[0].Base != after[0].Base {
				return fmt.Errorf("unknown English %q mismatch", before[0].Surface)
			}
			if c.Down != nil && (got.Accent.Down != *c.Down || c.Moras == nil || got.Accent.Moras != *c.Moras) {
				return fmt.Errorf("unknown English accent mismatch")
			}
			unknowns++
		case "frontend_unknown_accent":
			var a struct {
				Input []byte `json:"input"`
				Down  int    `json:"down"`
				Moras int    `json:"moras"`
			}
			if err := json.Unmarshal(raw, &a); err != nil {
				return err
			}
			input, err := codec.Decode(a.Input)
			if err != nil {
				return err
			}
			got, err := accent.Accent(input)
			if err != nil {
				return err
			}
			if got.Down != a.Down || got.Moras != a.Moras {
				return fmt.Errorf("normal unknown accent %q mismatch", input)
			}
			accents++
		case "frontend_english_transliteration", "frontend_english_spelling", "frontend_english_merge":
			if err := json.Unmarshal(raw, &e); err != nil {
				return err
			}
			if e.Event == "frontend_english_transliteration" {
				got, err := transliteration.Transliterate(e.Input)
				if err != nil {
					return err
				}
				pron, err := codec.Decode(e.Pronunciation)
				if err != nil {
					return err
				}
				if got != pron {
					return fmt.Errorf("normal transliteration %q got %q want %q", e.Input, got, pron)
				}
				conversions++
			} else if e.Event == "frontend_english_spelling" {
				got, err := rules.Spell(e.Input, accent)
				if err != nil {
					return err
				}
				pron, err := codec.Decode(e.Pronunciation)
				if err != nil {
					return err
				}
				if got.Pronunciation != pron || got.Accent.Down != e.Down {
					return fmt.Errorf("normal spelling %q mismatch", e.Input)
				}
				spellings++
			} else {
				input, err := decodeWords(e.Before)
				if err != nil {
					return err
				}
				want, err := decodeWords(e.After)
				if err != nil {
					return err
				}
				got, err := frontend.MergeEnglishRun(input, codec, func(s string) (frontend.UnknownKanaReading, error) {
					return frontend.CompileUnknownEnglish(s, rules, transliteration, accent, fallback)
				})
				if err != nil {
					return err
				}
				if len(got) != len(want) {
					return fmt.Errorf("English merge %d length mismatch", merges)
				}
				for i, w := range want {
					if got[i].Surface != w.Surface || got[i].Base != w.Base || got[i].POS != w.POS || got[i].Pronunciation != w.Pronunciation {
						return fmt.Errorf("English merge %d word %d got %+v want %+v", merges, i, got[i], w)
					}
				}
				merges++
			}
		}
	}
	if modelResults == 0 || spellings+conversions == 0 || merges == 0 {
		return fmt.Errorf("incomplete normal spelling trace")
	}
	fmt.Printf("Verified %d normal-engine spellings, %d transliterations, %d accents and %d English merge runs\n", spellings, conversions, accents, merges)
	fmt.Printf("Verified %d complete unknown English word conversions\n", unknowns)
	fmt.Printf("Verified %d English group scans (%d origin-bearing inputs, %d origin boundaries)\n", groups, originWords, boundaryWords)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
