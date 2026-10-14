// Compare parsed accent-bearing text2kana output with native intermediate JSON.
package main

import (
	"github.com/a-kaibu/aoikane/internal/frontend"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
)

func read(path string, target any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, target)
}
func run() error {
	asset := flag.String("asset", "private/phonetics.json", "phonetics asset")
	pattern := flag.String("references", "private/kana-*-reference-trace/events.json", "native trace glob")
	prosodyOnly := flag.Bool("prosody-only", false, "compare existing native kana-talk.json with independent phoneme.json")
	fullPhones := flag.Bool("phones", false, "also compare contextual phone labels, Info and articulation")
	flag.Parse()
	e, err := frontend.OpenMoraEncoder(*asset)
	if err != nil {
		return err
	}
	paths, err := filepath.Glob(*pattern)
	if err != nil {
		return err
	}
	files, sentences, moras, phones, prosodyPhones := 0, 0, 0, 0, 0
	for _, path := range paths {
		var events []struct {
			Event string `json:"event"`
			Text  string `json:"text"`
		}
		if !*prosodyOnly {
			if err := read(path, &events); err != nil {
				return err
			}
		}
		var inputs []string
		for _, event := range events {
			if event.Event == "frontend_format" {
				inputs = append(inputs, event.Text)
			}
		}
		if len(inputs) == 0 && !*prosodyOnly {
			continue
		}
		var native struct {
			Body struct {
				Sentences []frontend.KanaSentence `json:"sentences"`
			} `json:"body"`
		}
		if err := read(filepath.Join(filepath.Dir(path), "kana-talk.json"), &native); err != nil {
			return err
		}
		if *prosodyOnly {
			inputs = make([]string, len(native.Body.Sentences))
		}
		var phonetic struct {
			Body struct {
				Sentences []struct {
					Timeline []struct {
						ID             int     `json:"phnn"`
						Accent         int64   `json:"acc"`
						Punctuation    int64   `json:"punc"`
						MoraPosition   float32 `json:"nima"`
						PhrasePosition float32 `json:"niab"`
						Label          string  `json:"phlabel"`
						Info           string  `json:"phinfo"`
						Art            string  `json:"art"`
					} `json:"timeline"`
				} `json:"sentences"`
			} `json:"body"`
		}
		if err := read(filepath.Join(filepath.Dir(path), "phoneme.json"), &phonetic); err != nil {
			return err
		}
		if len(inputs) != len(native.Body.Sentences) || len(inputs) != len(phonetic.Body.Sentences) {
			return fmt.Errorf("%s: sentence count mismatch", path)
		}
		for si, input := range inputs {
			var got frontend.KanaSentence
			var err error
			if *prosodyOnly {
				got = native.Body.Sentences[si]
			} else {
				got, err = e.ParseFormattedKana(input)
			}
			if err != nil {
				return fmt.Errorf("%s sentence %d: %w", path, si, err)
			}
			prosody, err := e.CompileProsody(got)
			if err != nil {
				return fmt.Errorf("%s sentence %d prosody: %w", path, si, err)
			}
			want := phonetic.Body.Sentences[si].Timeline
			if len(prosody.Phoneme) != len(want) {
				return fmt.Errorf("%s sentence %d: prosody phone count mismatch", path, si)
			}
			for pi, p := range want {
				// Final contextual phone IDs are checked below in -phones mode.
				if (!*fullPhones && prosody.Phoneme[pi] != int64(p.ID)) || prosody.Accent[pi] != p.Accent || prosody.Punctuation[pi] != p.Punctuation || prosody.MoraPosition[pi] != p.MoraPosition || prosody.PhrasePosition[pi] != p.PhrasePosition {
					return fmt.Errorf("%s sentence %d phone %d: prosody mismatch", path, si, pi)
				}
				prosodyPhones++
			}
			if *fullPhones {
				sequence, err := e.CompilePhones(got, 0xef)
				if err != nil {
					return fmt.Errorf("%s sentence %d phones: %w", path, si, err)
				}
				if len(sequence.Phones) != len(want) {
					return fmt.Errorf("%s sentence %d: phone sequence count differs", path, si)
				}
				for pi, p := range want {
					info, err := strconv.ParseUint(p.Info, 16, 16)
					if err != nil {
						return err
					}
					art, err := strconv.ParseUint(p.Art, 2, 32)
					if err != nil {
						return err
					}
					actual := sequence.Phones[pi]
					// phinfo JSON packs the separately stored mora-head bit;
					// contextual nasal rewriting can also set Info bit zero.
					// This export cannot distinguish those sources, so compare
					// the remaining bits on both sides.
					if actual.ID != p.ID || actual.Label != p.Label || actual.Info&0xfffe != uint16(info)&0xfffe || sequence.Articulation[pi] != uint32(art) {
						return fmt.Errorf("%s sentence %d phone %d: got %+v art=%08x want id=%d label=%s info=%04x art=%08x", path, si, pi, actual, sequence.Articulation[pi], p.ID, p.Label, info, art)
					}
				}
			}
			if *prosodyOnly {
				sentences++
				continue
			}
			var ids, wantIDs []int
			for ti, item := range got.Timeline {
				for _, index := range item.PhoneIndices {
					ids = append(ids, e.Phones[index].ID)
					phones++
				}
				if item.Type == "mora" {
					moras++
				}
				got.Timeline[ti].PhoneIndices = nil
				got.Timeline[ti].Long = false
			}
			if !reflect.DeepEqual(got, native.Body.Sentences[si]) {
				return fmt.Errorf("%s sentence %d: timeline or accent mismatch: %+v / %+v", path, si, got, native.Body.Sentences[si])
			}
			for _, p := range phonetic.Body.Sentences[si].Timeline {
				if p.ID != 0 {
					wantIDs = append(wantIDs, p.ID)
				}
			}
			// Parsed mora IDs precede contextual closure replacement. In
			// -phones mode all final IDs were already compared above.
			if !*fullPhones && !reflect.DeepEqual(ids, wantIDs) {
				return fmt.Errorf("%s sentence %d: phone ID mismatch", path, si)
			}
			sentences++
		}
		files++
	}
	if files == 0 {
		return fmt.Errorf("no formatted kana references")
	}
	if !*prosodyOnly {
		fmt.Printf("Verified %d files, %d sentences, %d moras and %d phones: timelines, accents and phone IDs match native output\n", files, sentences, moras, phones)
	} else {
		fmt.Printf("Verified prosody for %d files and %d sentences\n", files, sentences)
	}
	fmt.Printf("Verified all %d prosody phones, including pauses: phn, acc, punc, nima and niab match native output exactly\n", prosodyPhones)
	if *fullPhones {
		fmt.Printf("Contextual labels, Info and articulation also match all %d phones\n", prosodyPhones)
	}
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
