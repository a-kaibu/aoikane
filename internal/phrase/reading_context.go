package phrase

import (
	"github.com/a-kaibu/aoikane/internal/frontend"
	"fmt"
	"sort"
)

// prepareNumeralChunks reconstructs BANCHA's 128370 prepass. It joins adjacent
// chunks by the classes of the two words at their boundary, before numeral
// replacement or reading selection. Original chunk ordinals remain available
// to contractNumeralChunks through the unmodified starts map.
func (f *Frontend) prepareNumeralChunks(words []frontend.Word, generated map[int]*Morph, starts map[int]bool) (map[int]bool, error) {
	if f.Numerals == nil {
		return starts, nil
	}
	classes := make([]int, len(words))
	for i, word := range words {
		var lw frontend.LanguageWord
		if m := generated[i]; m != nil {
			lw = frontend.LanguageWord{Surface: m.Surface, POS: m.POS}
		} else if !word.Entry.Unknown {
			var err error
			lw, err = f.Morphology.LanguageWord(word)
			if err != nil {
				return nil, err
			}
		}
		surface := fromCP932Form(lw.Surface)
		switch {
		case f.Numerals.Pre[surface] || lw.POS == 0x2030000:
			classes[i] = 1
		case f.Numerals.Signs[surface].Read != "":
			classes[i] = 2
		case f.Numerals.IsConnector(surface):
			classes[i] = 3
		case lw.POS == 0x1060000:
			classes[i] = 4
		case f.Numerals.Counters[surface].Read != "" || lw.POS == 0x1090700:
			classes[i] = 5
		case f.Numerals.Suffixes[surface]:
			classes[i] = 6
		}
	}
	prepared := make(map[int]bool, len(starts))
	for word, start := range starts {
		if !start || word <= 0 || word >= len(words) {
			prepared[word] = start
			continue
		}
		left, right := classes[word-1], classes[word]
		join := left == 1 && right >= 1 && right <= 4 ||
			left == 2 && right >= 2 && right <= 4 ||
			left == 3 && right == 4 ||
			left == 4 && right >= 3 && right <= 6 ||
			left == 5 && right == 6
		if !join {
			prepared[word] = true
		}
	}
	return prepared, nil
}

// readingChunks maps the post-numeral word vector to the original CaboCha
// chunks. S chunks retain their main chunk's ordinal and inherit its dependency,
// so splitting those ranges cannot change an any/all word predicate's result.
func readingChunks(units []unit, links, ordinals []int32) ([]frontend.ReadingChunk, []int, error) {
	if len(links) != len(ordinals) {
		return nil, nil, fmt.Errorf("reading chunk metadata lengths differ")
	}
	var chunks []frontend.ReadingChunk
	indices := make([]int, len(units))
	for i, u := range units {
		if i == 0 || u.chunkStart {
			if len(chunks) != 0 {
				chunks[len(chunks)-1].End = i
			}
			ordinal := len(chunks)
			if ordinal >= len(links) {
				return nil, nil, fmt.Errorf("reading chunk correspondence changed after numeral processing")
			}
			chunks = append(chunks, frontend.ReadingChunk{Begin: i, End: len(units), Ordinal: ordinals[ordinal], Subindex: -1, Link: links[ordinal]})
		}
		indices[i] = len(chunks) - 1
	}
	if len(chunks) != len(links) {
		return nil, nil, fmt.Errorf("reading chunk correspondence changed after numeral processing")
	}
	// N[0:0] with dependency -1 visits the real end-of-text morph, unlike
	// ordinary flat word ranges, which deliberately omit that sentinel.
	chunks = append(chunks, frontend.ReadingChunk{Begin: len(units), End: len(units) + 1, Ordinal: -1, Subindex: -1, Link: -1})
	return chunks, indices, nil
}

// contractNumeralChunks models the chunk removal caused by native word deletion
// (18fd40 -> 18fbe0). A consumed first word removes its chunk; the remaining
// ordinals stay unchanged. The 128370 prepass only redirects the absorbing
// chunk's own dependency when it points to the adjacent chunk being removed;
// dependencies from other chunks retain the removed ordinal.
func contractNumeralChunks(units []unit, wordIndices []int, wordCount int, starts map[int]bool, links []int32) ([]int32, []int32, error) {
	if len(units) != len(wordIndices) {
		return nil, nil, fmt.Errorf("numeral word correspondence differs")
	}
	if len(units) == 0 {
		return links, nil, nil
	}
	var firsts []int
	for word := range starts {
		firsts = append(firsts, word)
	}
	sort.Ints(firsts)
	if len(firsts) == 0 {
		return links, nil, nil
	}
	if links != nil && len(links) != len(firsts) {
		return nil, nil, fmt.Errorf("original chunk count differs")
	}
	wordOrdinal := make([]int32, wordCount)
	current := 0
	for word := range wordOrdinal {
		for current+1 < len(firsts) && word >= firsts[current+1] {
			current++
		}
		wordOrdinal[word] = int32(current)
	}
	var ordinals []int32
	live := map[int32]bool{}
	for i, word := range wordIndices {
		if word < 0 || word >= wordCount {
			return nil, nil, fmt.Errorf("numeral origin outside words")
		}
		ordinal := wordOrdinal[word]
		if i == 0 || units[i].chunkStart {
			if !live[ordinal] {
				ordinals = append(ordinals, ordinal)
				live[ordinal] = true
			} else {
				units[i].chunkStart = false
			}
		}
	}
	if links == nil {
		return nil, ordinals, nil
	}
	reduced := make([]int32, len(ordinals))
	for i, ordinal := range ordinals {
		end := int32(len(firsts))
		if i+1 < len(ordinals) {
			end = ordinals[i+1]
		}
		target := links[ordinal]
		for steps := 0; target > ordinal && target < end && !live[target]; steps++ {
			if int(target) >= len(links) || steps >= len(links) {
				return nil, nil, fmt.Errorf("invalid removed chunk dependency")
			}
			target = links[target]
		}
		reduced[i] = target
	}
	return reduced, ordinals, nil
}
