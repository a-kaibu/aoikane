package main

import (
	"github.com/a-kaibu/aoikane/internal/frontend"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func run() error {
	reference := flag.String("reference", "private/reading-accent-reference.json", "native counter and cache reference")
	trace := flag.String("trace", "", "optional normal synthesis trace")
	codecPath := flag.String("cp932", "private/cp932.json", "CP932 mappings")
	flag.Parse()
	codec, err := frontend.OpenCP932Codec(*codecPath)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(*reference)
	if err != nil {
		return err
	}
	var ref struct {
		Source   string `json:"source_sha256"`
		Counters []struct {
			Text  string `json:"text"`
			Moras int32  `json:"moras"`
		} `json:"counters"`
		Cases []struct {
			Previous frontend.ReadingAccent           `json:"previous"`
			Selected *frontend.PronunciationCandidate `json:"selected"`
			Result   frontend.ReadingAccent           `json:"result"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &ref); err != nil {
		return err
	}
	if ref.Source != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(ref.Counters) == 0 || len(ref.Cases) == 0 {
		return fmt.Errorf("invalid reading accent reference")
	}
	for i, c := range ref.Counters {
		got, err := frontend.CountAccentMoras(c.Text, codec)
		if err != nil {
			return err
		}
		if got != c.Moras {
			return fmt.Errorf("counter %d %q got %d want %d", i, c.Text, got, c.Moras)
		}
	}
	for i, c := range ref.Cases {
		got, err := frontend.UpdateReadingAccent(c.Previous, c.Selected, codec)
		if err != nil {
			return err
		}
		if got != c.Result {
			return fmt.Errorf("cache %d got %+v want %+v", i, got, c.Result)
		}
	}
	fmt.Printf("Verified %d mora counters and %d reading accent cache updates\n", len(ref.Counters), len(ref.Cases))
	if *trace == "" {
		return nil
	}
	data, err = os.ReadFile(*trace)
	if err != nil {
		return err
	}
	var events []json.RawMessage
	if err := json.Unmarshal(data, &events); err != nil {
		return err
	}
	count, parsed, models := 0, 0, 0
	for _, raw := range events {
		var h struct {
			Event string `json:"event"`
		}
		if err := json.Unmarshal(raw, &h); err != nil {
			return err
		}
		switch h.Event {
		case "frontend_error", "forward_error":
			return fmt.Errorf("failed trace")
		case "torch_jit_result":
			models++
		case "frontend_accent_parse":
			var e struct {
				Text    string   `json:"text"`
				Start   int      `json:"start"`
				Outputs []*int32 `json:"outputs"`
				Next    int      `json:"next"`
			}
			if err := json.Unmarshal(raw, &e); err != nil {
				return err
			}
			if len(e.Outputs) != 4 {
				return fmt.Errorf("missing accent parse outputs")
			}
			got, err := frontend.ParseAccentText(e.Text, e.Start)
			if err != nil {
				return err
			}
			values := [4]int32{got.First, got.Second, got.Third, got.Speaker}
			for i, p := range e.Outputs {
				if p != nil && *p != values[i] {
					return fmt.Errorf("normal accent parser output differs")
				}
			}
			if got.Next != e.Next {
				return fmt.Errorf("normal accent parser position differs")
			}
			parsed++
		case "frontend_reading_accent":
			var e struct {
				Previous []int32 `json:"previous"`
				Result   []int32 `json:"result"`
				Selected *struct {
					Pronunciation []byte `json:"pronunciation"`
					Accent        string `json:"accent"`
				} `json:"selected"`
			}
			if err := json.Unmarshal(raw, &e); err != nil {
				return err
			}
			if len(e.Previous) != 4 || len(e.Result) != 4 {
				return fmt.Errorf("invalid cache trace")
			}
			state := func(values []int32) frontend.ReadingAccent {
				return frontend.ReadingAccent{Up: values[0], Down: values[1], Moras: values[2], Speaker: values[3]}
			}
			var selected *frontend.PronunciationCandidate
			if e.Selected != nil {
				pron, err := codec.Decode(e.Selected.Pronunciation)
				if err != nil {
					return err
				}
				selected = &frontend.PronunciationCandidate{Pronunciation: pron, Accent: e.Selected.Accent}
			}
			got, err := frontend.UpdateReadingAccent(state(e.Previous), selected, codec)
			if err != nil {
				return err
			}
			if got != state(e.Result) {
				return fmt.Errorf("normal cache %d differs: %+v %+v", count, got, state(e.Result))
			}
			count++
		}
	}
	if count == 0 || parsed == 0 || models == 0 {
		return fmt.Errorf("incomplete reading accent trace")
	}
	fmt.Printf("Verified %d normal reading accent updates and %d accent text parses\n", count, parsed)
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
