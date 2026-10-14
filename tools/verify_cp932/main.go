package main

import (
	"github.com/a-kaibu/aoikane/internal/frontend"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func codeBytes(code int) []byte {
	if code > 255 {
		return []byte{byte(code >> 8), byte(code)}
	}
	return []byte{byte(code)}
}

func run() error {
	path := flag.String("table", "private/cp932.json", "Windows code page reference")
	trace := flag.String("trace", "", "optional native context events.json")
	flag.Parse()
	c, err := frontend.OpenCP932Codec(*path)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(*path)
	if err != nil {
		return err
	}
	var table struct {
		Encoding [][2]int `json:"encoding"`
		Decoding [][2]int `json:"decoding"`
	}
	if err := json.Unmarshal(b, &table); err != nil {
		return err
	}
	for _, entry := range table.Encoding {
		got, err := c.Encode(string(rune(entry[0])))
		if err != nil {
			return err
		}
		if !bytes.Equal(got, codeBytes(entry[1])) {
			return fmt.Errorf("encoding mismatch U+%04X", entry[0])
		}
	}
	for _, entry := range table.Decoding {
		got, err := c.Decode(codeBytes(entry[0]))
		if err != nil {
			return fmt.Errorf("decoding %04X: %w", entry[0], err)
		}
		if got != string(rune(entry[1])) {
			return fmt.Errorf("decoding mismatch %04X", entry[0])
		}
	}
	if _, err := c.Encode("😀"); err == nil {
		return fmt.Errorf("unrepresentable character accepted")
	}
	for _, bad := range [][]byte{{0x81}, {0x81, 0x30}} {
		if _, err := c.Decode(bad); err == nil {
			return fmt.Errorf("invalid CP932 accepted")
		}
	}
	count := 0
	if *trace != "" {
		b, err := os.ReadFile(*trace)
		if err != nil {
			return err
		}
		var events []json.RawMessage
		if err := json.Unmarshal(b, &events); err != nil {
			return err
		}
		for _, raw := range events {
			var header struct {
				Event string `json:"event"`
			}
			if err := json.Unmarshal(raw, &header); err != nil {
				return err
			}
			if header.Event == "frontend_error" || header.Event == "forward_error" {
				return fmt.Errorf("failed engine trace")
			}
			if header.Event != "frontend_context_build" {
				continue
			}
			var event struct {
				Words       [][]int `json:"words"`
				Result      []int   `json:"result"`
				SourceBytes int     `json:"source_bytes"`
			}
			if err := json.Unmarshal(raw, &event); err != nil {
				return err
			}
			surfaces := make([]string, len(event.Words))
			for i, word := range event.Words {
				b := make([]byte, len(word))
				for j, v := range word {
					if v < 0 || v > 255 {
						return fmt.Errorf("invalid trace byte")
					}
					b[j] = byte(v)
				}
				surfaces[i], err = c.Decode(b)
				if err != nil {
					return err
				}
				encoded, err := c.Encode(surfaces[i])
				if err != nil {
					return err
				}
				if !bytes.Equal(encoded, b) {
					return fmt.Errorf("native surface encoding mismatch")
				}
			}
			// Only source length affects this builder; the original string is
			// not present in context events. Reproduce that length explicitly.
			source := string(bytes.Repeat([]byte{'x'}, event.SourceBytes))
			got, err := c.ReadingContextText(source, surfaces)
			if err != nil {
				return err
			}
			encoded, err := c.Encode(got)
			if err != nil {
				return err
			}
			want := make([]byte, len(event.Result))
			for j, v := range event.Result {
				if v < 0 || v > 255 {
					return fmt.Errorf("invalid trace result")
				}
				want[j] = byte(v)
			}
			if !bytes.Equal(encoded, want) {
				return fmt.Errorf("native context text mismatch")
			}
			count++
		}
		if count == 0 {
			return fmt.Errorf("trace contains no context observations")
		}
	}
	fmt.Printf("Verified %d CP932 encodings, %d decodings, %d native contexts\n", len(table.Encoding), len(table.Decoding), count)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
