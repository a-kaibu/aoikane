package main

import (
	"github.com/a-kaibu/aoikane/internal/frontend"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"reflect"
)

func run() error {
	trace := flag.String("trace", "private/cleanup-group-final-trace/events.json", "normal cleanup trace")
	cp932 := flag.String("cp932", "private/cp932.json", "CP932 mapping")
	flag.Parse()
	codec, err := frontend.OpenCP932Codec(*cp932)
	if err != nil {
		return err
	}
	type nativeWord struct {
		Surface        []byte  `json:"surface"`
		Base           []byte  `json:"base"`
		Pronunciation  []byte  `json:"pronunciation"`
		POS            uint32  `json:"pos"`
		HasChildren    bool    `json:"has_children"`
		SelectedAccent *string `json:"selected_accent"`
	}
	decode := func(w *nativeWord) (frontend.CleanupWord, error) {
		if w == nil {
			return frontend.CleanupWord{}, nil
		}
		s, err := codec.Decode(w.Surface)
		if err != nil {
			return frontend.CleanupWord{}, err
		}
		b, err := codec.Decode(w.Base)
		if err != nil {
			return frontend.CleanupWord{}, err
		}
		p, err := codec.Decode(w.Pronunciation)
		if err != nil {
			return frontend.CleanupWord{}, err
		}
		return frontend.CleanupWord{KanaMergeWord: frontend.KanaMergeWord{Surface: s, Base: b, Pronunciation: p, POS: w.POS}, HasChildren: w.HasChildren}, nil
	}
	b, err := os.ReadFile(*trace)
	if err != nil {
		return err
	}
	var events []json.RawMessage
	if err := json.Unmarshal(b, &events); err != nil {
		return err
	}
	count, models, changed, removed, groups, groupRemoved := 0, 0, 0, 0, 0, 0
	for _, raw := range events {
		var e struct {
			Event           string        `json:"event"`
			Before          []*nativeWord `json:"before"`
			After           []*nativeWord `json:"after"`
			Keep            bool          `json:"keep"`
			NextHasChildren bool          `json:"next_has_children"`
			Moras           *int          `json:"next_accent_moras"`
			AccentText      *string       `json:"next_accent_text"`
		}
		// Decode only the header because other events use different array shapes.
		var h struct {
			Event string `json:"event"`
		}
		if err := json.Unmarshal(raw, &h); err != nil {
			return err
		}
		if h.Event == "frontend_error" || h.Event == "forward_error" {
			return fmt.Errorf("failed trace")
		}
		if h.Event == "torch_jit_result" {
			models++
		}
		if h.Event == "frontend_cleanup_group" {
			var group struct {
				Before          []nativeWord `json:"before"`
				After           []nativeWord `json:"after"`
				NextIsContainer bool         `json:"next_is_container"`
			}
			if err := json.Unmarshal(raw, &group); err != nil {
				return err
			}
			if !group.NextIsContainer || len(group.Before) == 0 {
				return fmt.Errorf("unsupported cleanup group boundaries")
			}
			convert := func(input []nativeWord) ([]frontend.CleanupGroupWord, error) {
				out := make([]frontend.CleanupGroupWord, len(input))
				for i := range input {
					word, err := decode(&input[i])
					if err != nil {
						return nil, err
					}
					out[i] = frontend.CleanupGroupWord{CleanupWord: word, SelectedAccent: input[i].SelectedAccent}
				}
				return out, nil
			}
			before, err := convert(group.Before)
			if err != nil {
				return err
			}
			original, err := convert(group.Before)
			if err != nil {
				return err
			}
			// The trace deliberately does not dereference the previous sentinel.
			// Its surface is irrelevant only when the first word is no separator.
			if before[0].Surface == "・" || before[0].Surface == "　" {
				return fmt.Errorf("missing previous boundary surface")
			}
			want, err := convert(group.After)
			if err != nil {
				return err
			}
			boundary := frontend.CleanupGroupWord{CleanupWord: frontend.CleanupWord{HasChildren: true}}
			got, err := frontend.CleanupWordGroup(before, frontend.CleanupGroupWord{}, boundary)
			if err != nil {
				return err
			}
			if !reflect.DeepEqual(got, want) {
				return fmt.Errorf("cleanup group %d differs\ngot %+v\nwant %+v", groups, got, want)
			}
			if !reflect.DeepEqual(before, original) {
				return fmt.Errorf("cleanup changed input")
			}
			groupRemoved += len(before) - len(got)
			groups++
		}
		if h.Event != "frontend_word_cleanup" {
			continue
		}
		if err := json.Unmarshal(raw, &e); err != nil {
			return err
		}
		if len(e.Before) != 3 || len(e.After) != 3 || e.Before[1] == nil || e.After[1] == nil {
			return fmt.Errorf("invalid cleanup neighbors")
		}
		var input [3]frontend.CleanupWord
		for i, w := range e.Before {
			input[i], err = decode(w)
			if err != nil {
				return err
			}
		}
		input[2].HasChildren = e.NextHasChildren
		var moras *int
		if e.Moras != nil {
			if e.AccentText == nil {
				return fmt.Errorf("missing native accent text")
			}
			record, err := frontend.ParseAccentText(*e.AccentText, 0)
			if err != nil {
				return err
			}
			value := int(record.Second)
			if value != *e.Moras {
				return fmt.Errorf("native selected accent value differs")
			}
			moras = &value
		}
		got, err := frontend.CleanupWordNeighbors(input[0], input[1], input[2], moras)
		if err != nil {
			return err
		}
		want, err := decode(e.After[1])
		if err != nil {
			return err
		}
		if got.Keep != e.Keep || got.Current.KanaMergeWord != want.KanaMergeWord || got.Current.HasChildren != want.HasChildren {
			return fmt.Errorf("cleanup %d %q differs", count, input[1].Surface)
		}
		if e.After[2] != nil {
			next, err := decode(e.After[2])
			if err != nil {
				return err
			}
			if got.Next.KanaMergeWord != next.KanaMergeWord {
				return fmt.Errorf("next word differs")
			}
		}
		if got.Current.Pronunciation != input[1].Pronunciation {
			changed++
		}
		if !got.Keep {
			removed++
		}
		count++
	}
	if count == 0 || models == 0 || groups == 0 {
		return fmt.Errorf("incomplete cleanup trace")
	}
	fmt.Printf("Verified %d native word cleanups (%d reading changes, %d removals)\n", count, changed, removed)
	if groups > 0 {
		fmt.Printf("Verified %d complete cleanup groups (%d removals)\n", groups, groupRemoved)
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
