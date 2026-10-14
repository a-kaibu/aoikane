// Compare full kana segmentation and phoneme IDs against native intermediate
// output; phoneme labels and prosody are deliberately checked by other stages.
package main

import (
	"github.com/a-kaibu/aoikane/internal/frontend"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	asset := flag.String("asset", "private/phonetics.json", "phonetic definitions")
	pattern := flag.String("references", "private/*/kana-talk.json", "native kana reference glob")
	flag.Parse()
	e, err := frontend.OpenMoraEncoder(*asset)
	if err != nil {
		return err
	}
	paths, err := filepath.Glob(*pattern)
	if err != nil {
		return err
	}
	files, sentences, moras, phones := 0, 0, 0, 0
	for _, path := range paths {
		var kana struct {
			Body struct {
				Sentences []struct {
					Timeline []struct {
						Type string `json:"type"`
						Mora string `json:"mora"`
					} `json:"timeline"`
				} `json:"sentences"`
			} `json:"body"`
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(b, &kana); err != nil {
			return err
		}
		var phoneme struct {
			Body struct {
				Sentences []struct {
					Timeline []struct {
						ID int `json:"phnn"`
					} `json:"timeline"`
				} `json:"sentences"`
			} `json:"body"`
		}
		b, err = os.ReadFile(filepath.Join(filepath.Dir(path), "phoneme.json"))
		if err != nil {
			return err
		}
		if err = json.Unmarshal(b, &phoneme); err != nil {
			return err
		}
		if len(kana.Body.Sentences) != len(phoneme.Body.Sentences) {
			return fmt.Errorf("%s: sentence count mismatch", path)
		}
		for si, s := range kana.Body.Sentences {
			var text strings.Builder
			var wantMoras []string
			for _, m := range s.Timeline {
				if m.Type == "mora" {
					text.WriteString(m.Mora)
					wantMoras = append(wantMoras, m.Mora)
				}
			}
			got, err := e.EncodePronunciation(text.String(), false)
			if err != nil {
				return fmt.Errorf("%s sentence %d: %w", path, si, err)
			}
			if len(got) != len(wantMoras) {
				return fmt.Errorf("%s sentence %d: mora count %d/%d", path, si, len(got), len(wantMoras))
			}
			var gotIDs, wantIDs []int
			for i, u := range got {
				if u.Text != wantMoras[i] {
					return fmt.Errorf("%s sentence %d mora %d: %q/%q", path, si, i, u.Text, wantMoras[i])
				}
				for _, index := range u.PhoneIndices {
					gotIDs = append(gotIDs, e.Phones[index].ID)
				}
			}
			for _, p := range phoneme.Body.Sentences[si].Timeline {
				if p.ID != 0 {
					wantIDs = append(wantIDs, p.ID)
				}
			}
			if len(gotIDs) != len(wantIDs) {
				return fmt.Errorf("%s sentence %d: phone count %d/%d", path, si, len(gotIDs), len(wantIDs))
			}
			for i, id := range gotIDs {
				if id != wantIDs[i] {
					return fmt.Errorf("%s sentence %d phone %d: ID %d/%d", path, si, i, id, wantIDs[i])
				}
			}
			sentences++
			moras += len(got)
			phones += len(gotIDs)
		}
		files++
	}
	if files == 0 || phones == 0 {
		return fmt.Errorf("no native mora references")
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]int{"files": files, "sentences": sentences, "moras": moras, "phones": phones})
}
