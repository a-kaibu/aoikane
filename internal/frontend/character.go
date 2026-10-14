package frontend

import (
	"encoding/binary"
	"fmt"
	"os"
	"unicode/utf8"
)

// CharacterInfo is the serialized SMecab character class. Category membership
// can overlap (for example 一 belongs to KANJI and KANJINUMERIC).
type CharacterInfo uint32

func (c CharacterInfo) Membership() uint32 { return uint32(c) & 0x3ffff }
func (c CharacterInfo) Category() int      { return int(uint32(c) >> 18 & 255) }
func (c CharacterInfo) PrefixLength() int  { return int(uint32(c) >> 26 & 15) }
func (c CharacterInfo) Group() bool        { return uint32(c)&0x40000000 != 0 }
func (c CharacterInfo) Invoke() bool       { return uint32(c)&0x80000000 != 0 }

type Characters struct {
	names []string
	table []byte
}

func OpenCharacters(path string) (*Characters, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) < 4 {
		return nil, fmt.Errorf("truncated character table")
	}
	n := int(binary.LittleEndian.Uint32(b))
	if n < 1 || n > 18 || len(b) != 4+n*32+65535*4 {
		return nil, fmt.Errorf("unsupported SMecab character table dimensions")
	}
	c := &Characters{table: b[4+n*32:]}
	for i := 0; i < n; i++ {
		raw := b[4+i*32 : 4+(i+1)*32]
		end := 0
		for end < len(raw) && raw[end] != 0 {
			end++
		}
		if end == 0 || end == len(raw) || !utf8.Valid(raw[:end]) {
			return nil, fmt.Errorf("invalid character category name")
		}
		c.names = append(c.names, string(raw[:end]))
	}
	return c, nil
}

func (c *Characters) Lookup(r rune) (CharacterInfo, error) {
	if r < 0 || r > utf8.MaxRune || r >= 0xd800 && r <= 0xdfff {
		return 0, fmt.Errorf("invalid Unicode scalar")
	}
	// The inspected engine uses a separate non-BMP category, not table[0].
	// U+FFFF is also beyond its 65535-entry table; its native behavior is not
	// yet verified, so do not read outside the asset or invent a class.
	if r == 0xffff {
		return 0, fmt.Errorf("U+FFFF character classification is not reconstructed")
	}
	if r > 0xffff {
		return CharacterInfo(0x2c0800), nil
	}
	return CharacterInfo(binary.LittleEndian.Uint32(c.table[int(r)*4:])), nil
}

func (c *Characters) CategoryName(info CharacterInfo) string {
	n := info.Category()
	if n >= len(c.names) {
		return ""
	}
	return c.names[n]
}

type UnknownCandidate struct {
	Text       string        `json:"text"`
	Info       CharacterInfo `json:"info"`
	Length     int           `json:"length"` // UTF-8 bytes, as in dictionary alternatives.
	Characters int           `json:"characters"`
	Context    uint16        `json:"context"`
	WordCost   int16         `json:"word_cost"`
}

// UnknownCandidates returns candidate creation order from native 1d5450:
// grouped run first, then prefixes, without repeating the grouped length.
// maxGroup is the caller's unknown-word group limit, not the category length.
// This only builds the morphological lattice; readings require later rules.
func (c *Characters) UnknownCandidates(text string, dictionaryMatched bool, maxGroup int) ([]UnknownCandidate, error) {
	if !utf8.ValidString(text) || len(text) > 1<<20 || maxGroup < 1 {
		return nil, fmt.Errorf("invalid unknown-word input or group limit")
	}
	if text == "" {
		return []UnknownCandidate{}, nil
	}
	r, width := utf8.DecodeRuneInString(text)
	info, err := c.Lookup(r)
	if err != nil {
		return nil, err
	}
	if dictionaryMatched && !info.Invoke() {
		return []UnknownCandidate{}, nil
	}
	costs := [12]int16{2338, 5059, 11218, 6765, 6912, 7016, 11140, 8697, 8026, 5059, 5059, 5059}
	category := info.Category()
	if category >= len(costs) {
		category = 0
	}
	makeCandidate := func(end, count int) UnknownCandidate {
		return UnknownCandidate{Text: text[:end], Info: info, Length: end, Characters: count, Context: 1912, WordCost: costs[category]}
	}
	var out []UnknownCandidate
	groupAdded := false
	groupEnd, groupCount := width, 1
	if info.Group() {
		for groupEnd < len(text) {
			r, w := utf8.DecodeRuneInString(text[groupEnd:])
			next, err := c.Lookup(r)
			if err != nil {
				return nil, err
			}
			if info.Membership()&next.Membership() == 0 {
				break
			}
			groupEnd += w
			groupCount++
		}
		if groupCount <= maxGroup {
			out = append(out, makeCandidate(groupEnd, groupCount))
			groupAdded = true
		}
	}
	end, count := width, 1
	for count <= info.PrefixLength() {
		if !groupAdded || end != groupEnd {
			out = append(out, makeCandidate(end, count))
		}
		if end == len(text) {
			break
		}
		r, w := utf8.DecodeRuneInString(text[end:])
		next, err := c.Lookup(r)
		if err != nil {
			return nil, err
		}
		if info.Membership()&next.Membership() == 0 {
			break
		}
		end += w
		count++
	}
	if len(out) == 0 {
		out = append(out, makeCandidate(width, 1))
	}
	return out, nil
}
