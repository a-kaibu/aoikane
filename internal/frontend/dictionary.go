// Package frontend reconstructs the original Japanese text frontend from its
// dictionary data. The dictionary reader is portable and does not use cgo.
package frontend

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// These offsets are the serialization headers in the inspected SMecab 1.6.1
// standard and kansai dictionaries, not the standard MeCab dictionary format.
const (
	trieHeader    = 37
	tokenHeader   = 50
	featureHeader = 37
	tokenBytes    = 9
)

type Dictionary struct {
	trie, tokens, features []byte
	count                  uint32
}

// EntryCount is the number of stored alternatives, including homographs.
func (d *Dictionary) EntryCount() uint32 { return d.count }

// LookupIndex reads a stored alternative independently of a trie match.
// Length is zero because no input text prefix has been matched.
func (d *Dictionary) LookupIndex(index uint32) (Entry, error) { return d.entry(index) }

type Accent struct {
	Down  int  `json:"down"`
	Moras int  `json:"moras"`
	Code  byte `json:"code"`
	// Up is explicitly stored by kaccent records. Ordinary accent records
	// only store Down/Moras; phrase processing determines their rise later.
	Up *int `json:"up,omitempty"`
}

// Entry retains alternative and conjugated readings without choosing a form.
// Context is the packed dictionary context ID. POS and conjugation type are
// stored before the surface string; Form is the token's conjugation form ID.
type Entry struct {
	Unknown           bool               `json:"unknown,omitempty"`
	Index             uint32             `json:"index"`
	Length            int                `json:"length"`
	Surface           string             `json:"surface"`
	Reading           string             `json:"reading"`
	Pronunciation     string             `json:"pronunciation"`
	Context           uint16             `json:"context"`
	WordCost          int16              `json:"word_cost"`
	Form              byte               `json:"form"`
	POS               byte               `json:"pos"`
	ConjugationType   byte               `json:"conjugation_type"`
	AccentCode        byte               `json:"accent_code"`
	Accents           []Accent           `json:"accents"`
	AccentConnections []AccentConnection `json:"accent_connections"`
	AccentGroups      []AccentGroup      `json:"accent_groups"`
}

func OpenDictionary(dir string) (*Dictionary, error) {
	d := new(Dictionary)
	for _, f := range []struct {
		name, magic string
		header      int
		target      *[]byte
	}{
		{"sys.dic", "\xffDar", trieHeader, &d.trie},
		{"sys.tok", "\xffTok", tokenHeader, &d.tokens},
		{"sys.fea", "\xffFea", featureHeader, &d.features},
	} {
		b, err := readDictionaryFile(filepath.Join(dir, f.name))
		if err != nil {
			return nil, err
		}
		if len(b) <= f.header || !bytes.HasPrefix(b, []byte(f.magic)) {
			return nil, fmt.Errorf("unsupported SMecab %s header", f.name)
		}
		if f.name == "sys.tok" {
			d.count = binary.LittleEndian.Uint32(b[46:50])
		}
		*f.target = b[f.header:]
	}
	if len(d.trie)%4 != 0 || uint64(len(d.tokens)) != uint64(d.count)*tokenBytes {
		return nil, fmt.Errorf("invalid SMecab dictionary dimensions")
	}
	return d, nil
}

func (d *Dictionary) unit(index uint32) (uint32, error) {
	if uint64(index)*4+4 > uint64(len(d.trie)) {
		return 0, fmt.Errorf("truncated dictionary trie")
	}
	return binary.LittleEndian.Uint32(d.trie[int(index)*4:]), nil
}

func offset(unit uint32) uint32 { return (unit >> 10) << ((unit & 512) >> 6) }

// LookupPrefix returns every dictionary spelling that prefixes text, including
// homographs. The trie uses the Darts-clone XOR transition representation:
// https://github.com/s-yata/darts-clone/blob/master/include/darts.h
func (d *Dictionary) LookupPrefix(text string) ([]Entry, error) {
	if !utf8.ValidString(text) {
		return nil, fmt.Errorf("text is not valid UTF-8")
	}
	var result []Entry
	var index uint32
	for i := 0; i < len(text); i++ {
		u, err := d.unit(index)
		if err != nil {
			return nil, err
		}
		index ^= offset(u) ^ uint32(text[i])
		u, err = d.unit(index)
		if err != nil {
			return nil, err
		}
		if u&0x800000ff != uint32(text[i]) {
			break
		}
		if u&256 == 0 {
			continue
		}
		leaf, err := d.unit(index ^ offset(u))
		if err != nil {
			return nil, err
		}
		first := leaf & 0x7fffffff
		if first >= d.count {
			return nil, fmt.Errorf("dictionary token index out of range")
		}
		packed := binary.LittleEndian.Uint16(d.tokens[int(first)*tokenBytes:])
		count := int(packed >> 13)
		// A zero count contributes one entry, then the following record is
		// checked. The first nonzero count describes all remaining entries.
		// Confirmed against the native lookup loop at RVA 1d4dd4..1d4dfe.
		for packed>>13 == 0 {
			count++
			if count > 256 || first+uint32(count) >= d.count {
				return nil, fmt.Errorf("invalid dictionary alternative count")
			}
			packed = binary.LittleEndian.Uint16(d.tokens[int(first+uint32(count))*tokenBytes:])
			if packed>>13 != 0 {
				count += int(packed >> 13)
				break
			}
		}
		for n := 0; n < count; n++ {
			e, err := d.entry(first + uint32(n))
			if err != nil {
				return nil, err
			}
			e.Length = i + 1
			result = append(result, e)
		}
	}
	return result, nil
}

func takeString(data []byte, at *int) ([]byte, error) {
	if *at < 0 || *at >= len(data) {
		return nil, fmt.Errorf("feature string out of range")
	}
	n := bytes.IndexByte(data[*at:], 0)
	if n < 0 || n > 65536 {
		return nil, fmt.Errorf("unterminated feature string")
	}
	s := data[*at : *at+n]
	*at += n + 1
	return s, nil
}

// Kana is a one-byte alphabet, with A1..FD mapped to Unicode 30A1..30FD.
// It is neither Shift JIS nor JIS X 0201 halfwidth katakana. ASCII suffixes
// encode conjugation alternatives and are preserved here.
func decodeKana(data []byte, previous string) (string, error) {
	if bytes.Equal(data, []byte{0xfe, 1}) {
		return previous, nil
	}
	var out strings.Builder
	for _, b := range data {
		switch {
		case b < 128:
			out.WriteByte(b)
		case b >= 0xa1 && b <= 0xfd:
			out.WriteRune(0x3000 + rune(b))
		default:
			return "", fmt.Errorf("unsupported dictionary kana byte %02x", b)
		}
	}
	return out.String(), nil
}

func (d *Dictionary) entry(index uint32) (Entry, error) {
	e := Entry{Index: index}
	if index >= d.count {
		return e, fmt.Errorf("dictionary alternative out of range")
	}
	b := d.tokens[int(index)*tokenBytes:]
	e.Context = binary.LittleEndian.Uint16(b) & 0x1fff
	e.WordCost = int16(binary.LittleEndian.Uint16(b[2:]))
	e.Form = b[4]
	at64 := uint64(binary.LittleEndian.Uint32(b[5:]))
	if at64 >= uint64(len(d.features)) {
		return e, fmt.Errorf("dictionary feature out of range")
	}
	at := int(at64)
	if len(d.features)-at < 2 {
		return e, fmt.Errorf("truncated dictionary POS and conjugation type")
	}
	e.POS, e.ConjugationType = d.features[at], d.features[at+1]
	at += 2
	surface, err := takeString(d.features, &at)
	if err != nil {
		return e, err
	}
	if !utf8.Valid(surface) {
		return e, fmt.Errorf("invalid dictionary surface UTF-8")
	}
	e.Surface = string(surface)
	reading, err := takeString(d.features, &at)
	if err != nil {
		return e, err
	}
	e.Reading, err = decodeKana(reading, e.Surface)
	if err != nil {
		return e, err
	}
	pron, err := takeString(d.features, &at)
	if err != nil {
		return e, err
	}
	e.Pronunciation, err = decodeKana(pron, e.Reading)
	if err != nil {
		return e, err
	}
	// Accent records can contain a linked list of alternatives, extra parameters
	// and concatenation rule IDs. The following POS record is separate.
	if at >= len(d.features) {
		return e, fmt.Errorf("truncated dictionary accent record")
	}
	e.AccentCode = d.features[at]
	e.AccentGroups, _, err = DecodeAccentGroups(d.features[at:])
	if err != nil {
		return e, err
	}
	e.Accents = e.AccentGroups[0].Accents
	e.AccentConnections = e.AccentGroups[0].Connections
	if e.AccentCode == 0 {
		e.Accents = nil
	}
	return e, nil
}

func readDictionaryFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err == nil || !os.IsNotExist(err) {
		return data, err
	}
	file, openErr := os.Open(path + ".gz")
	if openErr != nil {
		if os.IsNotExist(openErr) {
			return nil, err
		}
		return nil, openErr
	}
	defer file.Close()
	reader, readErr := gzip.NewReader(file)
	if readErr != nil {
		return nil, fmt.Errorf("read %s.gz: %w", path, readErr)
	}
	defer reader.Close()
	data, readErr = io.ReadAll(reader)
	if readErr != nil {
		return nil, fmt.Errorf("read %s.gz: %w", path, readErr)
	}
	return data, nil
}
