// Compare model articulation bitmaps against independently captured native
// intermediate output. No engine DLL or inference runtime is used.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/a-kaibu/aoikane/internal/frontend"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	asset := flag.String("asset", "private/phonetics.json", "private phonetics asset")
	pattern := flag.String("references", "private/*/phoneme.json", "native intermediate JSON glob")
	fromKana := flag.Bool("from-kana", false, "reconstruct phones from independent native kana instead of checking the earlier bitmap helper using final phone records")
	voicing := flag.Uint("voicing", 0xef, "native voicing configuration for kana reconstruction")
	flag.Parse()
	encoder, err := frontend.OpenMoraEncoder(*asset)
	if err != nil {
		return err
	}
	paths, err := filepath.Glob(*pattern)
	if err != nil {
		return err
	}
	count := 0
	for _, path := range paths {
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var doc struct {
			Body struct {
				Sentences []struct {
					Timeline []struct {
						ID    int    `json:"phnn"`
						Label string `json:"phlabel"`
						Info  string `json:"phinfo"`
						Art   string `json:"art"`
					} `json:"timeline"`
				} `json:"sentences"`
			} `json:"body"`
		}
		if err := json.Unmarshal(b, &doc); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		var kana struct {
			Body struct {
				Sentences []frontend.KanaSentence `json:"sentences"`
			} `json:"body"`
		}
		if *fromKana {
			b, err := os.ReadFile(filepath.Join(filepath.Dir(path), "kana-talk.json"))
			if err != nil {
				return err
			}
			if err := json.Unmarshal(b, &kana); err != nil {
				return err
			}
			if len(kana.Body.Sentences) != len(doc.Body.Sentences) {
				return fmt.Errorf("%s: kana/phoneme sentence count mismatch", path)
			}
		}
		for si, sentence := range doc.Body.Sentences {
			phones := make([]frontend.Allophone, len(sentence.Timeline))
			want := make([]uint32, len(phones))
			for i, p := range sentence.Timeline {
				info, err := strconv.ParseUint(p.Info, 16, 16)
				if err != nil {
					return fmt.Errorf("%s: phone info: %w", path, err)
				}
				art, err := strconv.ParseUint(p.Art, 2, 32)
				if err != nil || len(p.Art) != 32 {
					return fmt.Errorf("%s: invalid articulation bitmap", path)
				}
				phones[i] = frontend.Allophone{ID: p.ID, Label: p.Label, Info: uint16(info)}
				want[i] = uint32(art)
			}
			var got []uint32
			if *fromKana {
				// The closure pass copies the next stored bitmap. Inverting
				// final Info therefore cannot recover pre-pass articulation.
				sequence, err := encoder.CompilePhones(kana.Body.Sentences[si], uint32(*voicing))
				if err != nil {
					return fmt.Errorf("%s sentence %d: %w", path, si, err)
				}
				got = sequence.Articulation
			} else {
				got, err = encoder.ModelArticulation(phones)
			}
			if err != nil {
				return fmt.Errorf("%s sentence %d: %w", path, si, err)
			}
			if len(got) != len(want) {
				return fmt.Errorf("%s sentence %d: articulation count mismatch", path, si)
			}
			for i := range got {
				if got[i] != want[i] {
					return fmt.Errorf("%s sentence %d phone %d: got %08x want %08x", path, si, i, got[i], want[i])
				}
				count++
			}
		}
	}
	if count == 0 {
		return fmt.Errorf("no reference phonemes")
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]int{"files": len(paths), "phones": count})
}
