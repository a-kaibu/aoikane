// Package numeral reconstructs the engine's numeral reader (BANCHA pass 11f9c0):
// it turns a run of digit words and a counter into one word with its reading,
// pronunciation and accent groups, driven by the language's numeral rule file
// (bancharcx).
package numeral

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Word is a reading with its pronunciation (the "Sound" of the rule file).
type Word struct {
	Read, Sound string
}

// Digit is a digit word of the [数字] section.
type Digit struct {
	Word
	Type string // Jap, Eng, Unit or Con
	Jap  string // for Eng digits, the kanji digit
	Eng  string // for Eng digits, the English reading (ワン, ツー, ...)
}

// AccentWord is an entry of the [アクセント] table: a word with its accent
// (fall position and mora count) and the five flag digits.
type AccentWord struct {
	Sound, Read string
	Fall, Moras int
	Flag        string
}

// Counter is a counter (助数詞) of the [助数詞] section.
type Counter struct {
	Word
	Tn, Tw, Tl string
}

// Special is a [特殊読み] entry; Alternatives are tried in order.
type Special struct {
	Key, Display string
	Alternatives []SpecialReading
}

// SpecialReading is one alternative of a special reading.
type SpecialReading struct {
	Word
	Accent string // "accent=0-4:accent_con=*" without the key, e.g. "0-4" or "1-3,0-3"
	Con    string
}

// Rules is the parsed numeral rule file.
type Rules struct {
	ConsecutiveDigits map[string]bool
	FractionCounters  map[string]bool
	Digits            map[string]Digit
	Units             map[string]Digit
	Cons              map[string]Digit
	Tj                map[string]string
	Tk                map[string][]TkCode
	Accents           []AccentWord          // file order (longest first)
	BySound           map[string]AccentWord // first entry per Sound
	Counters          map[string]Counter
	Specials          map[string]Special
	Suffixes          map[string]bool
	Signs             map[string]Word
	Pre               map[string]bool
	// ZeroSkippingCounters are the literal counters of the 整数ゼロ飛ばし
	// pattern, which takes precedence over the leading-zero digit reading.
	ZeroSkippingCounters map[string]bool
	// ZeroRangeCounters are explicit range suffixes that allow leading zero.
	ZeroRangeCounters map[string]bool
	// Literal seconds use N'秒n/J'秒j. The fourth pattern field decides
	// whether a counter after the fractional digits is permitted.
	SecondsFractionPattern bool
	SecondsFractionCounter bool
	PointDecimalPrefixes   map[string]bool
	PostalPointPrefixes    map[string]bool
}

// TkCode is one value of a Tk row; Bang marks values preceded by '!'.
type TkCode struct {
	Value int
	Bang  bool
}

func attrs(fields []string) map[string]string {
	m := map[string]string{}
	for _, f := range fields {
		if i := strings.IndexByte(f, '='); i > 0 {
			m[f[:i]] = f[i+1:]
		}
	}
	return m
}

func stripComment(s string) string {
	if i := strings.Index(s, "//"); i >= 0 {
		s = s[:i]
	}
	if i := strings.Index(s, ";"); i >= 0 {
		s = s[:i]
	}
	return s
}

// ParseRules reads the bancharcx text (UTF-8).
func ParseRules(r io.Reader) (*Rules, error) {
	rules := &Rules{Digits: map[string]Digit{}, Units: map[string]Digit{}, Cons: map[string]Digit{},
		Tj: map[string]string{}, Tk: map[string][]TkCode{}, BySound: map[string]AccentWord{},
		Counters: map[string]Counter{}, Specials: map[string]Special{}, Suffixes: map[string]bool{},
		Signs: map[string]Word{}, Pre: map[string]bool{}, ZeroSkippingCounters: map[string]bool{}, ZeroRangeCounters: map[string]bool{},
		PointDecimalPrefixes: map[string]bool{}, PostalPointPrefixes: map[string]bool{}}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	section := ""
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if strings.HasPrefix(line, ";") || strings.TrimSpace(line) == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(strings.TrimSpace(line), "]") {
			section = strings.Trim(strings.TrimSpace(line), "[]")
			continue
		}
		fields := strings.Split(line, "\t")
		key := fields[0]
		switch section {
		case "パターン":
			pattern := strings.Split(line, ";")
			if len(pattern) > 3 && pattern[0] == "真分数=*" && pattern[2] == "n'／n/j'／j/" {
				if rules.FractionCounters == nil {
					rules.FractionCounters = map[string]bool{}
				}
				for _, literal := range strings.Split(pattern[3], "/") {
					if strings.HasPrefix(literal, "'") {
						rules.FractionCounters[strings.TrimPrefix(literal, "'")] = true
					}
				}
			}
			if len(pattern) > 3 && pattern[0] == "数字連続=*" && pattern[1] == "0" && pattern[3] == "@" {
				if rules.ConsecutiveDigits == nil {
					rules.ConsecutiveDigits = map[string]bool{}
				}
				for _, literal := range strings.Split(pattern[2], "/") {
					if strings.HasPrefix(literal, "'") {
						rules.ConsecutiveDigits[strings.TrimPrefix(literal, "'")] = true
					}
				}
			}
			if len(pattern) > 2 {
				var prefixes map[string]bool
				if strings.HasPrefix(pattern[0], "小数='") && strings.Contains(pattern[2], "n'・n") {
					prefixes = rules.PointDecimalPrefixes
				}
				if strings.HasPrefix(pattern[0], "電話='郵便番号/") && strings.Contains(pattern[2], "n'・n") {
					prefixes = rules.PostalPointPrefixes
				}
				if prefixes != nil {
					_, literals, _ := strings.Cut(pattern[0], "=")
					for _, literal := range strings.Split(literals, "/") {
						if strings.HasPrefix(literal, "'") && len(literal) > 1 {
							prefixes[strings.TrimPrefix(literal, "'")] = true
						}
					}
				}
			}
			if len(pattern) > 3 && strings.HasPrefix(pattern[0], "整数からＸ=*") &&
				(strings.Contains(pattern[2], "N'―N/") || strings.Contains(pattern[2], "J'―J/")) {
				for _, literal := range strings.Split(pattern[3], "/") {
					if strings.HasPrefix(literal, "'") {
						rules.ZeroRangeCounters[strings.TrimPrefix(literal, "'")] = true
					}
				}
			}
			if strings.HasPrefix(line, "時間=*;*;N'秒n/J'秒j/;") {
				parts := strings.Split(line, ";")
				rules.SecondsFractionPattern = true
				rules.SecondsFractionCounter = len(parts) > 3 && parts[3] == "*"
			}
			if strings.HasPrefix(line, "整数ゼロ飛ばし=") {
				parts := strings.Split(strings.TrimPrefix(line, "整数ゼロ飛ばし="), ";")
				if len(parts) >= 4 && parts[0] == "*" && parts[1] == "*" && parts[2] == "'〇j/'０n/" {
					for _, literal := range strings.Split(parts[3], "/") {
						if strings.HasPrefix(literal, "'") && len(literal) > 1 {
							rules.ZeroSkippingCounters[strings.TrimPrefix(literal, "'")] = true
						}
					}
				}
			}
		case "数字":
			a := attrs(fields[1:])
			switch {
			case a["Tj"] != "":
				rules.Tj[key] = a["Tj"]
			case strings.HasPrefix(strings.Join(fields[1:], "\t"), "Tk="):
				row, err := parseTk(strings.TrimPrefix(fields[1], "Tk="))
				if err != nil {
					return nil, fmt.Errorf("Tk row of %q: %w", key, err)
				}
				rules.Tk[key] = row
			default:
				d := Digit{Word: Word{Read: a["Read"], Sound: a["Sound"]}, Type: a["Type"], Jap: a["Jap"], Eng: a["Eng"]}
				switch d.Type {
				case "Jap", "Eng":
					rules.Digits[key] = d
				case "Unit":
					rules.Units[key] = d
				case "Con":
					rules.Cons[key] = d
				}
			}
		case "アクセント":
			a := attrs(fields[1:])
			var fall, moras int
			if _, err := fmt.Sscanf(strings.TrimSpace(stripComment(a["Accent"])), "%d-%d", &fall, &moras); err != nil {
				continue
			}
			w := AccentWord{Sound: key, Read: a["Read"], Fall: fall, Moras: moras, Flag: strings.TrimSpace(stripComment(a["Flag"]))}
			rules.Accents = append(rules.Accents, w)
			if _, ok := rules.BySound[key]; !ok {
				rules.BySound[key] = w
			}
		case "助数詞":
			a := attrs(fields[1:])
			rules.Counters[key] = Counter{Word: Word{Read: a["Read"], Sound: a["Sound"]}, Tn: a["Tn"], Tw: a["Tw"], Tl: strings.TrimSpace(stripComment(a["Tl"]))}
		case "特殊読み":
			if len(fields) < 5 {
				continue
			}
			a := attrs(fields[2:])
			sp := Special{Key: key, Display: fields[1]}
			reads, sounds, infos := splitAlt(a["Read"]), splitAlt(a["Sound"]), splitAlt(a["Info"])
			for i := range reads {
				alt := SpecialReading{Word: Word{Read: reads[i]}}
				if i < len(sounds) {
					alt.Sound = sounds[i]
				}
				if i < len(infos) {
					alt.Accent, alt.Con = parseInfo(infos[i])
				}
				sp.Alternatives = append(sp.Alternatives, alt)
			}
			rules.Specials[key] = sp
		case "接辞":
			rules.Suffixes[key] = true
		case "符号":
			a := attrs(fields[1:])
			rules.Signs[key] = Word{Read: a["Read"], Sound: a["Sound"]}
		case "前置助数詞":
			rules.Pre[key] = true
		}
	}
	return rules, sc.Err()
}

func parseTk(s string) ([]TkCode, error) {
	var out []TkCode
	bang := false
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == ' ' || c == '\t':
			bang = false
		case c == '!':
			bang = true
		case c >= '0' && c <= '9':
			out = append(out, TkCode{Value: int(c - '0'), Bang: bang})
			bang = false
		default:
			return nil, fmt.Errorf("unexpected %q", c)
		}
	}
	return out, nil
}

// splitAlt splits "{a/b/c}" (or a plain value) into its alternatives.
func splitAlt(v string) []string {
	v = strings.TrimSpace(stripComment(v))
	if v == "" {
		return nil
	}
	if strings.HasPrefix(v, "{") && strings.HasSuffix(v, "}") {
		return strings.Split(v[1:len(v)-1], "/")
	}
	return []string{v}
}

// parseInfo splits "accent=0-4:accent_con=*" into the accent and connection.
func parseInfo(s string) (accent, con string) {
	for _, p := range strings.Split(s, ":") {
		switch {
		case strings.HasPrefix(p, "accent="):
			accent = strings.TrimPrefix(p, "accent=")
		case strings.HasPrefix(p, "accent_con="):
			con = strings.TrimPrefix(p, "accent_con=")
		}
	}
	return
}
