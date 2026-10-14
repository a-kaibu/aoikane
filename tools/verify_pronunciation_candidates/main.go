// Compare complete reading records generated from Go dictionary entries with
// 1783a0 observations during normal authenticated synthesis.
package main

import (
	"github.com/a-kaibu/aoikane/internal/frontend"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"
)

func run() error {
	trace := flag.String("trace", "private/reading-candidate-trace/events.json", "normal candidate trace")
	directory := flag.String("dictionary", "private/frontend-data", "dictionary directory")
	codecPath := flag.String("cp932", "private/cp932.json", "CP932 mappings")
	selectionText := flag.String("selection-text", "", "also compare selected records from this original input text")
	morphologyPath := flag.String("morphology", "private/morphology.json", "morphology mappings")
	rulesPath := flag.String("rules", "private/reading-rules-utf8.txt", "reading selection rules")
	flag.Parse()
	dictionary, err := frontend.OpenDictionary(*directory)
	if err != nil {
		return err
	}
	codec, err := frontend.OpenCP932Codec(*codecPath)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(*trace)
	if err != nil {
		return err
	}
	var events []json.RawMessage
	if err := json.Unmarshal(data, &events); err != nil {
		return err
	}
	type key struct{ Reading, Pronunciation, Accent string }
	normalize := func(s string) string {
		return strings.Map(func(c rune) rune {
			if c >= '０' && c <= '９' {
				return c - '０' + '0'
			}
			return c
		}, s)
	}
	type binding struct {
		Entry frontend.Entry
		Only  int
	}
	entries := map[key]binding{}
	seen := map[uint32]bool{}
	for _, raw := range events {
		var e struct {
			Event string `json:"event"`
			Index uint32 `json:"index"`
			Count int    `json:"count"`
		}
		if err := json.Unmarshal(raw, &e); err != nil {
			return err
		}
		if e.Event == "frontend_error" || e.Event == "forward_error" {
			return fmt.Errorf("failed trace")
		}
		if e.Event != "dictionary_group" {
			continue
		}
		if e.Count < 1 || e.Count > 256 {
			return fmt.Errorf("invalid dictionary group")
		}
		for n := 0; n < e.Count; n++ {
			index := e.Index + uint32(n)
			if seen[index] {
				continue
			}
			seen[index] = true
			entry, err := dictionary.LookupIndex(index)
			if err != nil {
				return err
			}
			entries[key{normalize(entry.Reading), normalize(entry.Pronunciation), frontend.FormatAccentGroups(entry.AccentGroups)}] = binding{entry, -1}
			candidates, err := frontend.DictionaryCandidates(entry)
			if err != nil {
				return fmt.Errorf("entry %d: %w", index, err)
			}
			var readings, pronunciations, properties []string
			for i, c := range candidates {
				readings = append(readings, c.Reading)
				pronunciations = append(pronunciations, c.Pronunciation)
				properties = append(properties, "kaccent="+c.Accent+":kaccent_con="+c.Connection)
				entries[key{normalize(c.Reading), normalize(c.Pronunciation), "kaccent=" + c.Accent + ":kaccent_con=" + c.Connection}] = binding{entry, i}
			}
			wrap := func(values []string) string {
				text := strings.Join(values, "/")
				if len(values) > 1 {
					text = "{" + text + "}"
				}
				return text
			}
			entries[key{normalize(wrap(readings)), normalize(wrap(pronunciations)), wrap(properties)}] = binding{entry, -1}
		}
	}
	count, total, multi, models, placeholders := 0, 0, 0, 0, 0
	for _, raw := range events {
		var header struct {
			Event string `json:"event"`
		}
		if err := json.Unmarshal(raw, &header); err != nil {
			return err
		}
		if header.Event == "torch_jit_result" {
			models++
		}
		if header.Event != "word_property_candidates" {
			continue
		}
		var e struct {
			Event      string              `json:"event"`
			Fields     map[string][]byte   `json:"fields"`
			Candidates []map[string][]byte `json:"candidates"`
			Selected   int                 `json:"selected"`
		}
		if err := json.Unmarshal(raw, &e); err != nil {
			return err
		}
		decode := func(fields map[string][]byte) (frontend.PronunciationCandidate, error) {
			var c frontend.PronunciationCandidate
			for name, target := range map[string]*string{"reading": &c.Reading, "pronunciation": &c.Pronunciation, "accent": &c.Accent, "connection": &c.Connection} {
				var err error
				*target, err = codec.Decode(fields[name])
				if err != nil {
					return c, err
				}
			}
			return c, nil
		}
		input, err := decode(e.Fields)
		if err != nil {
			return err
		}
		bound, ok := entries[key{normalize(input.Reading), normalize(input.Pronunciation), input.Accent}]
		if !ok {
			// The native unknown-word initializer supplies this placeholder;
			// it is not a sys.tok entry or a generated unknown pronunciation.
			if input.Reading == "＊" && (input.Pronunciation == "＊" && input.Accent == "accent=*:accent_con=*" || input.Pronunciation == "" && input.Accent == "kaccent=*:kaccent_con=*") {
				placeholders++
				continue
			}
			return fmt.Errorf("unmatched candidate input %q %q %q", input.Reading, input.Pronunciation, input.Accent)
		}
		entry := bound.Entry
		got, err := frontend.DictionaryCandidates(entry)
		if err != nil {
			return err
		}
		if bound.Only >= 0 {
			got = got[bound.Only : bound.Only+1]
		}
		want := make([]frontend.PronunciationCandidate, len(e.Candidates))
		for i, c := range e.Candidates {
			want[i], err = decode(c)
			if err != nil {
				return err
			}
		}
		if e.Selected != 0 || !reflect.DeepEqual(got, want) {
			return fmt.Errorf("entry %d candidates differ\ngot %+v\nwant %+v", entry.Index, got, want)
		}
		if len(got) > 1 {
			multi++
		}
		total += len(got)
		count++
	}
	if count == 0 || multi == 0 || models == 0 {
		return fmt.Errorf("incomplete candidate trace")
	}
	fmt.Printf("Verified %d dictionary candidate constructions, %d records (%d multiple-candidate calls)\n", count, total, multi)
	fmt.Printf("Excluded %d native unknown-word placeholders\n", placeholders)
	if *selectionText != "" {
		analyzer, err := frontend.OpenAnalyzer(*directory)
		if err != nil {
			return err
		}
		morphology, err := frontend.OpenMorphology(*morphologyPath)
		if err != nil {
			return err
		}
		file, err := os.Open(*rulesPath)
		if err != nil {
			return err
		}
		defer file.Close()
		rules, err := frontend.ParseReadingRules(file)
		if err != nil {
			return err
		}
		normalized, err := frontend.NormalizeText(*selectionText)
		if err != nil {
			return err
		}
		words, err := analyzer.Analyze(normalized)
		if err != nil {
			return err
		}
		context := make([]frontend.LanguageWord, len(words))
		surfaces := make([]string, len(words))
		for i, w := range words {
			context[i], err = morphology.LanguageWord(w)
			if err != nil {
				return err
			}
			surfaces[i] = w.Text
		}
		contextText, err := codec.ReadingContextText(*selectionText, surfaces)
		if err != nil {
			return err
		}
		// Compare candidate-collapse calls, before document graph construction.
		collapsed := map[frontend.PronunciationCandidate]bool{}
		for _, raw := range events {
			var header struct {
				Event string `json:"event"`
			}
			if err := json.Unmarshal(raw, &header); err != nil {
				return err
			}
			if header.Event != "word_property_candidates" {
				continue
			}
			var event struct {
				Fields     map[string][]byte   `json:"fields"`
				Candidates []map[string][]byte `json:"candidates"`
			}
			if err := json.Unmarshal(raw, &event); err != nil {
				return err
			}
			if len(event.Candidates) != 1 {
				continue
			}
			inputReading, err := codec.Decode(event.Fields["reading"])
			if err != nil {
				return err
			}
			var candidate frontend.PronunciationCandidate
			for name, target := range map[string]*string{"reading": &candidate.Reading, "pronunciation": &candidate.Pronunciation, "accent": &candidate.Accent, "connection": &candidate.Connection} {
				*target, err = codec.Decode(event.Candidates[0][name])
				if err != nil {
					return err
				}
			}
			if inputReading == candidate.Reading && strings.HasPrefix(string(event.Fields["accent"]), "kaccent=") {
				collapsed[candidate] = true
			}
		}
		changed := 0
		for i, word := range words {
			candidate, selection, err := rules.SelectDictionaryCandidate(context, i, word.Entry, contextText)
			if err != nil {
				return err
			}
			if selection.Variant == 0 {
				continue
			}
			if !collapsed[candidate] {
				return fmt.Errorf("missing native selected candidate for %q: %+v", word.Text, candidate)
			}
			changed++
		}
		if changed == 0 {
			return fmt.Errorf("no nondefault selections to compare")
		}
		fmt.Printf("Verified %d nondefault reading selections with their complete candidate records\n", changed)
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
