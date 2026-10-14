package phrase

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

type RendakuTables struct {
	Source string `json:"rule_sha256"`
	Wa, On map[string]string
}

type RendakuAssets struct {
	Version          int
	DLL              string `json:"dll_sha256"`
	Capture          string `json:"capture_sha256"`
	LexiconSource    string `json:"lexicon_rule_sha256"`
	Languages        map[string]RendakuTables
	Kanji            map[string][4][]string
	Unvoiced, Voiced string
}

type RendakuRules struct {
	assets *RendakuAssets
	tables RendakuTables
}

func ParseRendakuAssets(reader io.Reader) (*RendakuAssets, error) {
	var a RendakuAssets
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&a); err != nil {
		return nil, err
	}
	var tail any
	if err := dec.Decode(&tail); err != io.EOF {
		return nil, fmt.Errorf("trailing rendaku asset")
	}
	if a.Version != 1 || a.DLL != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(a.Capture) != 64 || a.LexiconSource != "9a5ad3b3e2eb2059063b5c28afbf3b73163098ea82d523e1a8790faa4c6dc88c" || len(a.Kanji) != 6355 || a.Unvoiced != "カキクケコサシスセソタチツテトハヒフヘホ" || a.Voiced != "ガギグゲゴザジズゼゾダヂヅデドバビブベボパピプペポ" {
		return nil, fmt.Errorf("unsupported original rendaku asset")
	}
	return &a, nil
}

func (a *RendakuAssets) Rules(language string) (*RendakuRules, error) {
	tables, ok := a.Languages[language]
	if !ok || len(tables.Source) != 64 || len(tables.Wa) == 0 || len(tables.On) == 0 {
		return nil, fmt.Errorf("missing %s rendaku rules", language)
	}
	return &RendakuRules{assets: a, tables: tables}, nil
}

// onPrefix reconstructs 129590's first-character lookup and prefix comparison.
func (r *RendakuRules) prefix(surface, reading string, kind int) bool {
	first, _ := utf8.DecodeRuneInString(surface)
	for _, candidate := range r.assets.Kanji[string(first)][kind] {
		if strings.HasPrefix(reading, candidate) {
			return true
		}
	}
	return false
}

func (r *RendakuRules) voiced(text string) bool {
	return strings.ContainsAny(text, r.assets.Voiced)
}

func rendakuMoras(text string) int {
	n := 0
	for _, ch := range text {
		if startsMora(ch) {
			n++
		}
	}
	return n
}

func rendakuReading(word *Morph) string {
	if candidate := word.SelectedCandidate(); candidate != nil {
		return candidate.Reading
	}
	return ""
}

// precedingOn follows 1173b0's exact length/reading match against the final
// kanji's on-reading. A missing or nonmatching entry permits the previous word.
func (r *RendakuRules) precedingOn(previous *Morph) bool {
	surface := []rune(previous.Surface)
	if len(surface) == 0 {
		return false
	}
	reading := rendakuReading(previous)
	last := r.assets.Kanji[string(surface[len(surface)-1])][0]
	for _, candidate := range last {
		if reading != candidate {
			continue
		}
		moras := rendakuMoras(reading)
		return moras != 1 && (moras != 2 || !r.voiced(reading))
	}
	return true
}

// applies reconstructs 117720, including Lyman's law and the original POS gates.
func (r *RendakuRules) applies(word, previous *Morph) bool {
	if previous == nil {
		return false
	}
	reading := rendakuReading(word)
	first, size := utf8.DecodeRuneInString(reading)
	if !strings.ContainsRune(r.assets.Unvoiced, first) || r.voiced(reading[size:]) {
		return false
	}
	major := previous.POS & 0xff000000
	if major == 0x08000000 || major == 0x09000000 || major == 0x0b000000 || previous.POS == 0x06000000 || previous.POS == 0x07000000 || major == 0x02000000 || major == 0x03000000 && word.POS&0xff000000 == 0x03000000 || word.POS == 0x01060000 || word.POS&0xffff0000 == 0x01010000 {
		return false
	}
	if strings.HasPrefix(word.Surface, "先") || strings.HasPrefix(word.Surface, "服") {
		return false
	}
	if r.prefix(word.Surface, reading, 1) {
		return r.tables.Wa[word.Surface] == reading
	}
	return r.prefix(word.Surface, reading, 0) && r.precedingOn(previous) && r.tables.On[word.Surface] == reading
}

// ApplyRendaku follows 116ab0 on the post-cut chunks. The first morph of every
// D/S chunk has no predecessor for this pass. Replacement preserves accent and
// connection while retaining only the selected, voiced candidate (1783a0).
func (d *Document) ApplyRendaku(r *RendakuRules) error {
	if r == nil {
		return nil
	}
	for _, sentence := range d.Sentences {
		end := sentence.Next.Chunk
		for chunk := sentence.Chunk; chunk != end; chunk = chunk.Next {
			if chunk == nil || chunk.Next == nil {
				return fmt.Errorf("unterminated rendaku chunk")
			}
			var previous *Morph
			for word := chunk.First; word != chunk.Next.First; word = word.Next {
				if word == nil {
					return fmt.Errorf("unterminated rendaku word range")
				}
				if r.applies(word, previous) {
					if word.Selected < 0 || int(word.Selected) >= len(word.Cands) {
						return fmt.Errorf("invalid rendaku candidate")
					}
					candidate := word.Cands[word.Selected]
					first, size := utf8.DecodeRuneInString(candidate.Reading)
					pron, pronSize := utf8.DecodeRuneInString(candidate.Pronunciation)
					if first == pron {
						index := strings.IndexRune(r.assets.Unvoiced, first) / 3
						voiced := []rune(r.assets.Voiced)[index]
						candidate.Reading = string(voiced) + candidate.Reading[size:]
						candidate.Pronunciation = string(voiced) + candidate.Pronunciation[pronSize:]
						word.Cands, word.Selected = []Candidate{candidate}, 0
					}
				}
				previous = word
			}
		}
	}
	return nil
}
