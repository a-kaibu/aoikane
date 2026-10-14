package frontend

import (
	"fmt"
	"strings"
)

// PronunciationCandidate keeps a dictionary reading alternative and its
// corresponding accent and connection property together. Accent and Connection
// are the normalized native runtime strings, without inspection headers.
type PronunciationCandidate struct {
	Reading       string `json:"reading"`
	Pronunciation string `json:"pronunciation"`
	Accent        string `json:"accent"`
	Connection    string `json:"connection"`
}

// DictionaryCandidates connects the decoded dictionary groups to the runtime
// reading records produced by 1783a0. It does not apply phrase accent rules.
func DictionaryCandidates(entry Entry) ([]PronunciationCandidate, error) {
	if entry.Unknown {
		return nil, fmt.Errorf("unknown word needs a generated reading")
	}
	// Stored dictionary alternatives are parallel slash-separated values.
	// Braced prefix/suffix templates used by other native callers are separate.
	if strings.ContainsAny(entry.Reading+entry.Pronunciation, "{}") {
		return nil, fmt.Errorf("dictionary reading template is not reconstructed")
	}
	readings, pronunciations := strings.Split(entry.Reading, "/"), strings.Split(entry.Pronunciation, "/")
	if len(readings) != len(pronunciations) || len(readings) != len(entry.AccentGroups) {
		return nil, fmt.Errorf("dictionary candidate counts differ: readings %d pronunciations %d accents %d", len(readings), len(pronunciations), len(entry.AccentGroups))
	}
	if len(readings) > 256 {
		return nil, fmt.Errorf("too many pronunciation candidates")
	}
	out := make([]PronunciationCandidate, len(readings))
	for i, group := range entry.AccentGroups {
		accents, err := NormalizeAccent(group.Accents)
		if err != nil {
			return nil, fmt.Errorf("candidate %d: %w", i, err)
		}
		reading := strings.Map(func(c rune) rune {
			if c >= '0' && c <= '9' {
				return c - '0' + '０'
			}
			return c
		}, readings[i])
		// 1952d0 with no allowed ASCII set and flag 0 retains katakana
		// through ヴ and the long mark. It discards separators and suffixes.
		pron := strings.Map(func(c rune) rune {
			if c >= 0x30a1 && c <= 0x30f4 || c == 0x30fc {
				return c
			}
			return -1
		}, pronunciations[i])
		accent := strings.TrimPrefix(FormatAccentRecord(accents), "kaccent=")
		connection := strings.TrimPrefix(FormatAccentConnections(group.Connections, true), "kaccent_con=")
		out[i] = PronunciationCandidate{Reading: reading, Pronunciation: pron, Accent: accent, Connection: connection}
	}
	return out, nil
}

// SelectDictionaryCandidate selects the complete record with the same variant
// index as the reading rules. It never independently picks an accent variant.
func (r *ReadingRules) SelectDictionaryCandidate(words []LanguageWord, current int, entry Entry, contextText string) (PronunciationCandidate, ReadingSelection, error) {
	candidates, err := DictionaryCandidates(entry)
	if err != nil {
		return PronunciationCandidate{}, ReadingSelection{}, err
	}
	readings := make([]string, len(candidates))
	for i, c := range candidates {
		readings[i] = c.Reading
	}
	selection, err := r.SelectReadingWithContextText(words, current, readings, contextText)
	if err != nil {
		return PronunciationCandidate{}, ReadingSelection{}, err
	}
	if selection.Variant < 0 || selection.Variant >= len(candidates) {
		return PronunciationCandidate{}, ReadingSelection{}, fmt.Errorf("selected dictionary candidate is missing")
	}
	return candidates[selection.Variant], selection, nil
}
