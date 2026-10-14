package frontend

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// NormalizeText implements the valid UTF-8 character normalization in 1e3c70.
// It is distinct from Unicode NFKC and from the later CP932 conversion.
func NormalizeText(text string) (string, error) {
	if len(text) > 1024*1024 || !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
		return "", fmt.Errorf("invalid frontend text")
	}
	var out strings.Builder
	for len(text) > 0 {
		first, size := utf8.DecodeRuneInString(text)
		text = text[size:]
		if first >= 1 && first <= 31 || first >= 127 && first <= 159 {
			continue
		}
		switch first {
		case ' ':
			first = '\u3000'
		case '"':
			first = '\u201d'
		case '\'':
			first = '\u2019'
		case '\\':
			first = '\uffe5'
		default:
			if first >= 33 && first <= 126 {
				first += 0xff01 - 33
			} else if first >= 0xff71 && first <= 0xff9d {
				first = []rune("アイウエオカキクケコサシスセソタチツテトナニヌネノハヒフヘホマミムメモヤユヨラリルレロワン")[first-0xff71]
			} else if first >= 0xff61 && first <= 0xff70 {
				first = []rune("。「」、・ヲァィゥェォャュョッー")[first-0xff61]
			}
		}
		next, nextSize := utf8.DecodeRuneInString(text)
		if combined, ok := combineVoicing(first, next); ok {
			first = combined
			text = text[nextSize:]
		}
		switch first {
		case '-', '\u2010', '\u2011', '\u2212', '\uff0d':
			first = '\u2212'
		case '\u2012', '\u2013', '\u2014', '\u2015':
			first = '\u2015'
		case '\u301c', '\uff5e':
			first = '\uff5e'
		}
		out.WriteRune(first)
	}
	return out.String(), nil
}

func combineVoicing(first, next rune) (rune, bool) {
	semi := next == 0x309a || next == 0x309c || next == 0xff9f
	voiced := next == 0x3099 || next == 0x309b || next == 0xff9e
	if !semi && !voiced {
		return first, false
	}
	for _, base := range []rune{0x30cf, 0x306f} {
		if first >= base && first <= base+12 && (first-base)%3 == 0 {
			if semi {
				return first + 2, true
			}
			return first + 1, true
		}
	}
	if semi {
		return first, false
	}
	for _, interval := range [][2]rune{{0x30ab, 0x30c1}, {0x30c4, 0x30c8}, {0x304b, 0x3061}, {0x3064, 0x3068}} {
		if first >= interval[0] && first <= interval[1] && (first-interval[0])%2 == 0 {
			return first + 1, true
		}
	}
	if first == 0x30a6 {
		return 0x30f4, true
	}
	if first == 0x3046 {
		return 0x3094, true
	}
	return first, false
}
