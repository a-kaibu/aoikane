package frontend

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

// CP932Codec uses the exported Windows code page rather than an approximate
// Shift-JIS mapping. Encoding never substitutes an unrepresentable character.
type CP932Codec struct {
	encode map[rune]uint16
	decode map[uint16]rune
}

func OpenCP932Codec(path string) (*CP932Codec, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) > 4*1024*1024 {
		return nil, fmt.Errorf("CP932 table exceeds limit")
	}
	var table struct {
		CodePage int      `json:"code_page"`
		Encoding [][2]int `json:"encoding"`
		Decoding [][2]int `json:"decoding"`
	}
	if err := json.Unmarshal(b, &table); err != nil {
		return nil, err
	}
	if table.CodePage != 932 || len(table.Encoding) == 0 || len(table.Decoding) == 0 {
		return nil, fmt.Errorf("invalid CP932 table")
	}
	c := &CP932Codec{encode: make(map[rune]uint16), decode: make(map[uint16]rune)}
	for _, entry := range table.Encoding {
		r, code := entry[0], entry[1]
		if r < 0 || r > 65535 || r >= 0xd800 && r <= 0xdfff || code < 0 || code > 65535 {
			return nil, fmt.Errorf("invalid CP932 encoding")
		}
		if _, exists := c.encode[rune(r)]; exists {
			return nil, fmt.Errorf("duplicate CP932 encoding")
		}
		c.encode[rune(r)] = uint16(code)
	}
	for _, entry := range table.Decoding {
		code, r := entry[0], entry[1]
		if code < 0 || code > 65535 || r < 0 || r > 65535 || r >= 0xd800 && r <= 0xdfff {
			return nil, fmt.Errorf("invalid CP932 decoding")
		}
		if _, exists := c.decode[uint16(code)]; exists {
			return nil, fmt.Errorf("duplicate CP932 decoding")
		}
		c.decode[uint16(code)] = rune(r)
	}
	// the dictionary spells the minus sign U+2212; Windows maps it to the same
	// code as U+FF0D
	if _, ok := c.encode[0x2212]; !ok {
		c.encode[0x2212] = c.encode[0xff0d]
	}
	return c, nil
}

func (c *CP932Codec) Encode(text string) ([]byte, error) {
	if !utf8.ValidString(text) || len(text) > 1024*1024 {
		return nil, fmt.Errorf("invalid CP932 source text")
	}
	var result []byte
	for _, r := range text {
		code, ok := c.encode[r]
		if !ok {
			return nil, fmt.Errorf("character U+%04X is not representable in CP932", r)
		}
		if code > 255 {
			result = append(result, byte(code>>8))
		}
		result = append(result, byte(code))
	}
	return result, nil
}

func (c *CP932Codec) Decode(data []byte) (string, error) {
	if len(data) > 1024*1024 {
		return "", fmt.Errorf("CP932 input exceeds limit")
	}
	var result strings.Builder
	for i := 0; i < len(data); i++ {
		code := uint16(data[i])
		if data[i] >= 0x81 && data[i] <= 0x9f || data[i] >= 0xe0 && data[i] <= 0xfc {
			if i+1 >= len(data) {
				return "", fmt.Errorf("truncated CP932 character")
			}
			i++
			code = code<<8 | uint16(data[i])
		}
		r, ok := c.decode[code]
		if !ok {
			return "", fmt.Errorf("invalid CP932 code %04X", code)
		}
		result.WriteRune(r)
	}
	return result.String(), nil
}

// ReadingContextText encodes the pre-processing word surfaces and preserves
// the original UTF-8 input length used by the inspected normal engine path.
func (c *CP932Codec) ReadingContextText(source string, surfaces []string) (string, error) {
	if !utf8.ValidString(source) || strings.ContainsRune(source, 0) || len(source) > 1024*1024 || len(surfaces) > 8192 {
		return "", fmt.Errorf("invalid reading context source")
	}
	words := make([][]byte, len(surfaces))
	for i, surface := range surfaces {
		var err error
		words[i], err = c.Encode(surface)
		if err != nil {
			return "", err
		}
	}
	b, err := BuildReadingContext(words, len(source))
	if err != nil {
		return "", err
	}
	return c.Decode(b)
}
