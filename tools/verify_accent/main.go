// Compare the portable accent record decoder with native reference output.
package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/a-kaibu/aoikane/internal/frontend"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	path := flag.String("reference", "private/accent-reference.json", "native accent reader reference")
	flag.Parse()
	b, err := os.ReadFile(*path)
	if err != nil {
		return err
	}
	var ref struct {
		Source string `json:"source_sha256"`
		Cases  []struct {
			Data     string `json:"data"`
			Consumed int    `json:"consumed"`
			Text     string `json:"text"`
			Source   string `json:"source"`
		} `json:"cases"`
		Connections []struct {
			Data     string `json:"data"`
			Consumed int    `json:"consumed"`
			Text     string `json:"text"`
			Source   string `json:"source"`
			Kana     bool   `json:"kana"`
		} `json:"connections"`
		Groups []struct {
			Data     string `json:"data"`
			Consumed int    `json:"consumed"`
			Text     string `json:"text"`
			Source   string `json:"source"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(b, &ref); err != nil {
		return err
	}
	if ref.Source != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(ref.Cases) == 0 {
		return fmt.Errorf("missing or incompatible accent references")
	}
	variants := 0
	for i, c := range ref.Cases {
		data, err := hex.DecodeString(c.Data)
		if err != nil {
			return err
		}
		records, consumed, err := frontend.DecodeAccentRecord(data)
		if err != nil {
			return fmt.Errorf("case %d %s: %w", i, c.Source, err)
		}
		text := frontend.FormatAccentRecord(records)
		if consumed != c.Consumed || text != c.Text {
			return fmt.Errorf("case %d %s: bytes %d/%d, text %q/%q", i, c.Source, consumed, c.Consumed, text, c.Text)
		}
		variants += len(records)
		// A binary prefix that cuts off an alternative must fail rather than
		// reading the following feature record or accepting partial accents.
		for end := 0; end < consumed; end++ {
			if _, _, err := frontend.DecodeAccentRecord(data[:end]); err == nil {
				return fmt.Errorf("case %d accepts truncated prefix of %d bytes", i, end)
			}
		}
	}
	if len(ref.Connections) == 0 {
		return fmt.Errorf("no connection references")
	}
	connections := 0
	for i, c := range ref.Connections {
		data, err := hex.DecodeString(c.Data)
		if err != nil {
			return err
		}
		records, consumed, err := frontend.DecodeAccentConnections(data)
		if err != nil {
			return fmt.Errorf("connection %d %s: %w", i, c.Source, err)
		}
		text := frontend.FormatAccentConnections(records, c.Kana)
		if consumed != c.Consumed || text != c.Text {
			return fmt.Errorf("connection %d %s: bytes %d/%d, text %q/%q", i, c.Source, consumed, c.Consumed, text, c.Text)
		}
		connections += len(records)
		for end := 0; end < consumed; end++ {
			if _, _, err := frontend.DecodeAccentConnections(data[:end]); err == nil {
				return fmt.Errorf("connection %d accepts truncated prefix of %d bytes", i, end)
			}
		}
	}
	if len(ref.Groups) == 0 {
		return fmt.Errorf("no pronunciation group references")
	}
	groups := 0
	for i, c := range ref.Groups {
		data, err := hex.DecodeString(c.Data)
		if err != nil {
			return err
		}
		records, consumed, err := frontend.DecodeAccentGroups(data)
		if err != nil {
			return fmt.Errorf("group %d %s: %w", i, c.Source, err)
		}
		text := frontend.FormatAccentGroups(records)
		if consumed != c.Consumed || text != c.Text {
			return fmt.Errorf("group %d %s: bytes %d/%d, text %q/%q", i, c.Source, consumed, c.Consumed, text, c.Text)
		}
		groups += len(records)
		for end := 0; end < consumed; end++ {
			if _, _, err := frontend.DecodeAccentGroups(data[:end]); err == nil {
				return fmt.Errorf("group %d accepts truncated prefix of %d bytes", i, end)
			}
		}
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]int{"cases": len(ref.Cases), "alternatives": variants, "connection_cases": len(ref.Connections), "connections": connections, "group_cases": len(ref.Groups), "pronunciation_groups": groups})
}
