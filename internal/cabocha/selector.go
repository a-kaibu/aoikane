// Copyright (c) 2001-2008, Taku Kudo. See CABOCHA-LICENSE.
package cabocha

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Token contains decoded CaboCha token fields, before dependency selection.
// Features are the comma-separated IPA fields, including empty fields.
type Token struct {
	Surface    string
	Normalized string
	Features   []string
}

type ChunkSpan struct{ Start, Size int }

// Selection contains decoded Selector features and positions within its chunk.
type Selection struct {
	ChunkSpan
	Head, Func int
	Features   []string
}

var ipaFunctionPrefixes = []string{"助詞", "助動詞", "動詞,非自立", "動詞,接尾", "形容詞,非自立", "形容詞,接尾"}

func matchesPrefix(feature string, patterns []string) bool {
	for _, p := range patterns {
		if strings.HasPrefix(feature, p) {
			return true
		}
	}
	return false
}
func tokenField(t Token, index int) (string, bool) {
	if index >= len(t.Features) || t.Features[index] == "*" {
		return "", false
	}
	return t.Features[index], true
}

// SelectIPA follows CaboCha's IPA Selector, retaining feature order and empty
// conjugation fields. Input spans must partition the token sequence.
func SelectIPA(tokens []Token, spans []ChunkSpan) ([]Selection, error) {
	result := make([]Selection, len(spans))
	next := 0
	for i, span := range spans {
		if span.Start != next || span.Size <= 0 || span.Start < 0 || span.Start > len(tokens) || span.Size > len(tokens)-span.Start {
			return nil, fmt.Errorf("invalid CaboCha chunk span")
		}
		next += span.Size
		selected := Selection{ChunkSpan: span}
		add := func(feature string) { selected.Features = append(selected.Features, feature) }
		for j := 0; j < span.Size; j++ {
			token := tokens[span.Start+j]
			feature := strings.Join(token.Features, ",")
			surface := token.Normalized
			if surface == "。" || surface == "、" {
				add("GPUNC:" + surface)
				add("FPUNC:" + surface)
			}
			if utf8.RuneCountInString(surface) == 1 && strings.Contains("(（‘“《「『［〈｛", surface) {
				add("GOB:" + surface)
				add("FOB:" + surface)
				add("GOB:1")
				add("FOB:1")
			}
			if utf8.RuneCountInString(surface) == 1 && strings.Contains(")）’”》」』］〉｝", surface) {
				add("GCB:" + surface)
				add("FCB:" + surface)
				add("GCB:1")
				add("FCB:1")
			}
			if strings.HasPrefix(feature, "助詞") {
				add("FCASE:" + surface)
			}
			functional := matchesPrefix(feature, ipaFunctionPrefixes)
			if functional {
				selected.Func = j
			}
			if !functional && !strings.HasPrefix(feature, "記号") {
				selected.Head = j
			}
		}
		if selected.Head > selected.Func {
			selected.Func = selected.Head
		}
		head, fun := tokens[span.Start+selected.Head], tokens[span.Start+selected.Func]
		emit := func(prefix string, token Token) {
			add(prefix + "S:" + token.Normalized)
			for k := 0; k < 4 && k < len(token.Features); k++ {
				if token.Features[k] == "*" {
					break
				}
				add(prefix + "P" + strconv.Itoa(k) + ":" + token.Features[k])
			}
			if cform, ok := tokenField(token, 5); ok {
				add(prefix + "F:" + cform)
			}
		}
		emit("FH", head)
		emit("FF", fun)
		emit("FL", tokens[span.Start])
		emit("FR", tokens[next-1])
		add("LF:" + fun.Normalized)
		add("RL:" + tokens[span.Start].Normalized)
		add("RH:" + head.Normalized)
		add("RF:" + fun.Normalized)
		if i == 0 {
			add("FBOS:1")
		}
		if i == len(spans)-1 {
			add("FEOS:1")
		}
		feature := strings.Join(fun.Features, ",")
		if strings.HasPrefix(feature, "助詞") {
			add("GCASE:" + fun.Normalized)
		}
		if matchesPrefix(feature, []string{"助詞", "副詞", "連体詞", "接続詞"}) {
			add("A:" + fun.Normalized)
		} else if cform, ok := tokenField(fun, 5); ok {
			add("A:" + cform)
		} else {
			fields := fun.Features
			if len(fields) > 4 {
				fields = fields[:4]
			}
			for k, field := range fields {
				if field == "*" {
					fields = fields[:k]
					break
				}
			}
			add("A:" + strings.Join(fields, "-"))
		}
		// The original serializes features separated by ASCII spaces, then
		// tokenizes at those spaces (retaining empty fields), at most 128 entries.
		selected.Features = strings.Split(strings.Join(selected.Features, " "), " ")
		if len(selected.Features) > 128 {
			selected.Features = selected.Features[:128]
		}
		result[i] = selected
	}
	if next != len(tokens) {
		return nil, fmt.Errorf("CaboCha chunks do not cover tokens")
	}
	return result, nil
}
