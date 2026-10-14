package frontend

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// Analyzer builds a dictionary and unknown-word lattice. Unknown-word readings,
// form-dependent pronunciation, and phrase rules still require later stages.
type Analyzer struct {
	Dictionary  *Dictionary
	Characters  *Characters
	matrix      []byte
	left, right int
}

type Word struct {
	Begin      int             `json:"begin"`
	End        int             `json:"end"`
	Text       string          `json:"text"`
	Entry      Entry           `json:"entry"`
	Morphology *WordMorphology `json:"morphology,omitempty"`
	// AccentGroups retain every pronunciation alternative, with native rise
	// positions filled in. Choosing and connecting groups is a later stage.
	AccentGroups []AccentGroup `json:"accent_groups,omitempty"`
}

func OpenAnalyzer(dir string) (*Analyzer, error) {
	d, err := OpenDictionary(dir)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(filepath.Join(dir, "matrix.bin"))
	if err != nil {
		return nil, err
	}
	if len(b) < 4 {
		return nil, fmt.Errorf("truncated connection matrix")
	}
	l, r := int(binary.LittleEndian.Uint16(b)), int(binary.LittleEndian.Uint16(b[2:]))
	if l == 0 || r == 0 || uint64(l)*uint64(r)*2+4 != uint64(len(b)) {
		return nil, fmt.Errorf("invalid connection matrix dimensions")
	}
	characters, err := OpenCharacters(filepath.Join(dir, "char.bin"))
	if err != nil {
		return nil, err
	}
	return &Analyzer{Dictionary: d, Characters: characters, matrix: b[4:], left: l, right: r}, nil
}

func (a *Analyzer) transition(previous, current uint16) (int64, error) {
	if int(previous) >= a.left || int(current) >= a.right {
		return 0, fmt.Errorf("dictionary context outside connection matrix")
	}
	i := (int(previous) + int(current)*a.left) * 2
	return int64(int16(binary.LittleEndian.Uint16(a.matrix[i:]))), nil
}

type pathNode struct {
	word     Word
	cost     int64
	order    uint64
	previous *pathNode
}

// Analyze finds the minimum dictionary and connection cost path. Positions are
// UTF-8 byte offsets, matching the original dictionary's search contract.
func (a *Analyzer) Analyze(text string) ([]Word, error) {
	if !utf8.ValidString(text) || len(text) > 1<<20 {
		return nil, fmt.Errorf("text must be valid UTF-8 and at most 1 MiB")
	}
	if text == "" {
		return []Word{}, nil
	}
	ends := make([]map[uint16]*pathNode, len(text)+1)
	ends[0] = map[uint16]*pathNode{0: {}}
	var order uint64
	for pos := 0; pos < len(text); {
		_, width := utf8.DecodeRuneInString(text[pos:])
		if len(ends[pos]) == 0 {
			pos += width
			continue
		}
		entries, err := a.Dictionary.LookupPrefix(text[pos:])
		if err != nil {
			return nil, fmt.Errorf("at byte %d: %w", pos, err)
		}
		unknown, err := a.Characters.UnknownCandidates(text[pos:], len(entries) > 0, 24)
		if err != nil {
			return nil, fmt.Errorf("at byte %d: %w", pos, err)
		}
		// Both dictionary lookup (1d4cd0) and unknown generation (1d1fe0)
		// prepend candidates. Assemble creation order, then visit it backwards
		// like 1e2f90. Unknown readings are filled by later frontend stages.
		for _, u := range unknown {
			entries = append(entries, Entry{Unknown: true, Length: u.Length, Surface: u.Text, Context: u.Context, WordCost: u.WordCost})
		}
		for i := len(entries) - 1; i >= 0; i-- {
			entry := entries[i]
			end := pos + entry.Length
			if end > len(text) {
				return nil, fmt.Errorf("dictionary match exceeds input")
			}
			best := int64(math.MaxInt64)
			var prev *pathNode
			for context, node := range ends[pos] {
				c, err := a.transition(context, entry.Context)
				if err != nil {
					return nil, err
				}
				cost := node.cost + c + int64(entry.WordCost)
				// Native 1e2600 breaks total-cost ties by predecessor cost;
				// 1e2f90 prepends ending nodes, so equal predecessors use
				// the most recently inserted node.
				if cost < best || (cost == best && (prev == nil || node.cost < prev.cost || node.cost == prev.cost && node.order > prev.order)) {
					best, prev = cost, node
				}
			}
			if ends[end] == nil {
				ends[end] = map[uint16]*pathNode{}
			}
			if old := ends[end][entry.Context]; old == nil || best <= old.cost {
				order++
				ends[end][entry.Context] = &pathNode{word: Word{Begin: pos, End: end, Text: text[pos:end], Entry: entry}, cost: best, order: order, previous: prev}
			}
		}
		pos += width
	}
	best := int64(math.MaxInt64)
	var last *pathNode
	for context, node := range ends[len(text)] {
		c, err := a.transition(context, 0)
		if err != nil {
			return nil, err
		}
		if cost := node.cost + c; cost < best || (cost == best && (last == nil || node.cost < last.cost || node.cost == last.cost && node.order > last.order)) {
			best, last = cost, node
		}
	}
	if last == nil {
		return nil, fmt.Errorf("no complete morphological path")
	}
	var words []Word
	for n := last; n.previous != nil; n = n.previous {
		word := n.word
		for _, group := range word.Entry.AccentGroups {
			accents, err := NormalizeAccent(group.Accents)
			if err != nil {
				return nil, fmt.Errorf("word at byte %d: %w", word.Begin, err)
			}
			word.AccentGroups = append(word.AccentGroups, AccentGroup{Accents: accents, Connections: group.Connections})
		}
		words = append(words, word)
	}
	for l, r := 0, len(words)-1; l < r; l, r = l+1, r-1 {
		words[l], words[r] = words[r], words[l]
	}
	return words, nil
}

// DictionaryPronunciation exposes the first stored pronunciation form for
// inspection. Form and phrase rules still have to be applied by the frontend.
func DictionaryPronunciation(words []Word) string {
	var b strings.Builder
	for _, w := range words {
		b.WriteString(strings.Split(w.Entry.Pronunciation, "/")[0])
	}
	return b.String()
}
