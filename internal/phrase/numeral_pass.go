package phrase

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/a-kaibu/aoikane/internal/frontend"
	"github.com/a-kaibu/aoikane/internal/numeral"
)

// numeralRun is a run of digit words, with its optional counter, that the
// numeral pass (11f9c0) merges into one word.
type numeralRun struct {
	start, end int // words[start:end]
	number     string
	counter    string
	// surface is the text of all merged words (with separators).
	surface  string
	readings []numeral.Reading
}

// 120780 requires the selected native noun-number POS (0x1060000), not
// merely a surface present in the numeral table. For example, 京 can be a
// place name キョー and must retain its dictionary reading in that case.
func (f *Frontend) isNumeralWord(w frontend.Word) bool {
	if !f.Numerals.IsNumberWord(w.Text) || w.Entry.Unknown {
		return false
	}
	if f.Morphology == nil {
		return true // standalone scanner tests without dictionary POS metadata
	}
	lw, err := f.Morphology.LanguageWord(w)
	return err == nil && lw.POS == 0x1060000
}

// scanNumber collects the digit and unit words from i (a decimal point between
// digits and, for plain digit strings, thousands separators belong to it) and
// returns the end index and the text.
func (f *Frontend) scanNumber(words []frontend.Word, i int, starts map[int]bool, wholeRun bool) (int, string) {
	j := i
	var sb strings.Builder
	for j < len(words) && (j == i || wholeRun || !starts[j] || words[j-1].Text == "．") {
		if f.isNumeralWord(words[j]) {
			sb.WriteString(words[j].Text)
			j++
			continue
		}
		// a decimal point between digits belongs to the number
		if words[j].Text == "．" && j > i && j+1 < len(words) && f.isNumeralWord(words[j+1]) {
			sb.WriteString("．")
			j++
			continue
		}
		break
	}
	return j, sb.String()
}

// digitsOnly reports a string of full-width digits.
func digitsOnly(s string) bool {
	for _, r := range s {
		if r < '０' || r > '９' {
			return false
		}
	}
	return s != ""
}

// thousands joins "1,234,567" (a separator followed by exactly three digits)
// into one digit string; it returns the new end and text.
func (f *Frontend) thousands(words []frontend.Word, j int, text string) (int, string) {
	if !digitsOnly(text) || utf8.RuneCountInString(text) > 3 {
		return j, text
	}
	for j+1 < len(words) && words[j].Text == "，" {
		end := j + 1
		var group string
		for end < len(words) && digitsOnly(words[end].Text) && utf8.RuneCountInString(group) < 4 {
			group += words[end].Text
			end++
		}
		if end != j+1+3 || !digitsOnly(group) || utf8.RuneCountInString(group) != 3 {
			break
		}
		text += group
		j = end
	}
	return j, text
}

// counterAfter reads an optional counter word at j.
func (f *Frontend) counterAfter(words []frontend.Word, j int, starts map[int]bool) (counter string, fall, moras int, end int) {
	end = j
	if j < len(words) && (!starts[j] || words[j-1].Text == "．") && !words[j].Entry.Unknown {
		if _, ok := f.Numerals.Counters[words[j].Text]; ok {
			if cands, err := frontend.DictionaryCandidates(words[j].Entry); err == nil && len(cands) > 0 {
				var up int
				fmt.Sscanf(cands[0].Accent, "%d-%d-%d", &up, &fall, &moras)
				return words[j].Text, fall, moras, j + 1
			}
		}
	}
	return "", 0, 0, j
}

// findNumeralRuns locates the runs the reader can handle: numbers with an
// optional counter, decimals, thousands separators, ranges (～, 対), fractions
// times, hyphenated numbers, slash dates, phone numbers and parenthesized
// series. Runs next to an unsupported numeric connector are left alone.
func (f *Frontend) findNumeralRuns(words []frontend.Word, starts map[int]bool) []numeralRun {
	rules := f.Numerals
	var runs []numeralRun
	text := func(a, b int) string {
		var sb strings.Builder
		for _, w := range words[a:b] {
			sb.WriteString(w.Text)
		}
		return sb.String()
	}
	for i := 0; i < len(words); {
		j, number := i, ""
		if words[i].Text == "．" && i+1 < len(words) && f.isNumeralWord(words[i+1]) {
			j, number = f.scanNumber(words, i+1, starts, false)
			// Written middle points can follow the leading full stop.
			for j+1 < len(words) && words[j].Text == "・" && f.isNumeralWord(words[j+1]) {
				e, tail := f.scanNumber(words, j+1, starts, true)
				number, j = number+"．"+tail, e
			}
			// A single leading point is not part of a number. A second
			// point selects the native generic point/digit stream.
			if !strings.Contains(number, "．") {
				i++
				continue
			}
			if mixedNumberScripts(strings.Split(number, "．"), rules) {
				// Mixed scripts use the ordinary generic numeric path;
				// native excludes the initial point from that word.
				i++
				continue
			}
			number = "．" + number
		} else if f.isNumeralWord(words[i]) {
			j, number = f.scanNumber(words, i, starts, false)
		} else {
			i++
			continue
		}
		j, number = f.thousands(words, j, number)
		// a decimal fraction after thousands separators (1,234.5)
		if digitsOnly(number) && j+1 < len(words) && words[j].Text == "．" && j > i+1 && words[i+1].Text == "，" {
			if e, frac := f.scanNumber(words, j+1, starts, true); digitsOnly(frac) {
				number, j = number+"．"+frac, e
			}
		}
		run := numeralRun{start: i, number: number}
		blockedBefore := i > 1 && rules.IsConnector(words[i-1].Text) && f.isNumeralWord(words[i-2])
		// A connector-shaped counter already consumed by the previous run
		// belongs to that run. 秒 in 零秒八億 cannot block reading 八億.
		if len(runs) > 0 {
			previous := runs[len(runs)-1]
			if previous.end == i && previous.counter != "" {
				blockedBefore = false
			}
		}
		// patterns joining numbers
		if j+1 < len(words) && !blockedBefore {
			conn := words[j].Text
			switch conn {
			case "～", "対", "／", "：", "−", "ー", "―", "（", "）", "・", "点":
				parts, end := []string{number}, j
				var separators []string
				for end+1 < len(words) && (words[end].Text == conn ||
					(conn == "（" || conn == "）") && (words[end].Text == "（" || words[end].Text == "）") ||
					streamConnector(conn) && streamConnector(words[end].Text)) {
					if words[end].Text != conn && conn == "／" && utf8.RuneCountInString(parts[0]) == 4 && f.validDate(parts) {
						break
					}
					e, next := f.scanNumber(words, end+1, starts, true)
					e, next = f.thousands(words, e, next)
					if next == "" {
						break
					}
					separators = append(separators, words[end].Text)
					parts, end = append(parts, next), e
				}
				if len(parts) > 1 {
					if r, n, ok := f.patternStream(separators, parts, words, i, end, starts); ok {
						run.readings, run.end, run.counter = r, n, ""
						run.surface = fromCP932Form(text(i, n))
						runs = append(runs, run)
						i = n
						continue
					}
				}
			}
		}
		// 何, 数 and 幾 alone keep their dictionary readings (何か, 数人 ...)
		indefinite := number == "何" || number == "数" || number == "幾"
		req := numeral.Request{Number: number}
		counter, fall, moras, end := f.counterAfter(words, j, starts)
		if counter != "" {
			req.Counter, req.CounterFall, req.CounterMoras, req.CounterHasInfo = counter, fall, moras, true
			run.counter = counter
		}
		j = end
		run.end = j
		if counter == "秒" && j < len(words) && f.isNumeralWord(words[j]) {
			e, tail := f.scanNumber(words, j, starts, true)
			last := numeral.Request{Number: tail}
			c, cf, cm, ce := f.counterAfter(words, e, starts)
			if c != "" {
				last.Counter, last.CounterFall, last.CounterMoras, last.CounterHasInfo = c, cf, cm, true
				e = ce
			}
			firstTail, firstLast, firstEnd, firstCounter := tail, last, e, c
			seconds := []numeral.Request{req}
			for last.Counter == "秒" && e < len(words) && f.isNumeralWord(words[e]) {
				seconds = append(seconds, last)
				e, tail = f.scanNumber(words, e, starts, true)
				last = numeral.Request{Number: tail}
				c, cf, cm, ce = f.counterAfter(words, e, starts)
				if c != "" {
					last.Counter, last.CounterFall, last.CounterMoras, last.CounterHasInfo = c, cf, cm, true
					e = ce
				}
			}
			if len(seconds) > 1 {
				// The generic repeated-seconds stream retains comma boundaries.
				// Keep later digits in the same numeral rather than treating
				// them as independent dictionary words across chunk starts.
				for last.Counter == "" && e+1 < len(words) && words[e].Text == "，" && f.isNumeralWord(words[e+1]) {
					var next string
					e, next = f.scanNumber(words, e+1, starts, true)
					last.Number += "，" + next
					c, cf, cm, ce = f.counterAfter(words, e, starts)
					if c != "" {
						last.Counter, last.CounterFall, last.CounterMoras, last.CounterHasInfo = c, cf, cm, true
						e = ce
					}
				}
				if readings, err := rules.ReadSecondsSeries(seconds, last); err == nil {
					run.readings, run.end, run.surface, run.counter = readings, e, text(i, e), c
					runs = append(runs, run)
					i = e
					continue
				}
				// An unsupported complete stream must not let the old pair
				// fallback consume later words while skipping their numbers.
				tail, last, e, c = firstTail, firstLast, firstEnd, firstCounter
			}
			fractional := rules.SecondsFractionPattern && !mixedNumberScripts([]string{number, tail}, rules) &&
				(c == "" || rules.SecondsFractionCounter)
			for _, ch := range tail {
				if _, ok := rules.Digits[string(ch)]; !ok {
					fractional = false
				}
			}
			if readings, err := rules.ReadSeconds(req, last, fractional); err == nil {
				run.readings, run.end, run.surface, run.counter = readings, e, text(i, e), c
				runs = append(runs, run)
				i = e
				continue
			}
		}
		// a connector next to the run only matters when a number continues on its
		// other side (3-5, 12:30); brackets and the like do not block
		// Once a counter has been consumed, the following connector starts a
		// separate numeric expression (native 11f9c0); it cannot block this run.
		blocked := blockedBefore || (counter == "" && j+1 < len(words) && rules.IsConnector(words[j].Text) && f.isNumeralWord(words[j+1]))
		if !blocked && !(indefinite && run.counter == "") {
			read := rules.Read
			if i > 0 && run.counter == "" && englishPrefixes[words[i-1].Text] {
				read = func(r numeral.Request) ([]numeral.Reading, error) {
					if rd, err := rules.ReadEnglish(r.Number); err == nil {
						return rd, nil
					}
					return rules.Read(r)
				}
			}
			if readings, err := read(req); err == nil {
				run.readings = readings
				run.surface = text(i, j)
				runs = append(runs, run)
			} else if strings.HasPrefix(number, "．") {
				// A unit stream outside 1043f0's digit grammar leaves the
				// initial point out and retries the ordinary numeric path.
				i++
				continue
			}
		}
		i = j
	}
	return runs
}

// pattern reads a connector pattern over the numbers it joins (and for ranges
// an optional counter after the last). It returns the readings and the end.
func (f *Frontend) pattern(conn string, parts []string, words []frontend.Word, begin, end int, starts map[int]bool) ([]numeral.Reading, int, bool) {
	rules := f.Numerals
	last := numeral.Request{Number: parts[len(parts)-1]}
	counter, fall, moras, e := f.counterAfter(words, end, starts)
	if counter != "" {
		last.Counter, last.CounterFall, last.CounterMoras, last.CounterHasInfo = counter, fall, moras, true
	}
	a, b := parts[0], parts[len(parts)-1]
	if conn == "・" || conn == "点" {
		if counter != "" {
			end = e
		}
		mixed := mixedNumberScripts(parts, rules)
		if conn == "・" {
			postal := numeralPrefix(words, begin, rules.PostalPointPrefixes)
			phone := phonePrefix(words, begin)
			if (phone || postal && len(parts) == 2 && !mixed) && (counter == "" || counter == "番" || counter == "号") {
				r, err := rules.ReadPrefixTelephone(parts, last)
				return r, end, err == nil
			}
			if postal {
				last.NoSpecial = true
				r, err := rules.ReadSeries(parts[:len(parts)-1], last)
				return r, end, err == nil
			}
			if !mixed && counter == "" && len(parts) <= 3 && numeralPrefix(words, begin, rules.PointDecimalPrefixes) && (pointFraction(b, rules) || japaneseNumberParts(parts, rules)) {
				if r, err := rules.ReadLabelDecimal(parts, last); err == nil {
					return r, end, true
				}
			}
		}
		// Mixed digit scripts use the generic stream even when the next
		// fragment starts with a written unit; the point-unit pattern
		// requires a uniform digit script across the whole expression.
		if conn == "・" && mixed {
			last.NoSpecial = true
			r, err := rules.ReadSeries(parts[:len(parts)-1], last)
			return r, end, err == nil
		}
		if conn == "・" && len(parts) == 2 {
			chars := []rune(b)
			if len(chars) > 0 {
				if _, unit := rules.Units[string(chars[0])]; unit {
					last.NoSpecial = true
					var rd []numeral.Reading
					var err error
					// The address pattern excludes a literal leading 〇/０.
					// Its fallback keeps the written pause instead of inserting ノ.
					if counter != "" || numeralPrefix(words, begin, rules.PointDecimalPrefixes) || strings.HasPrefix(a, "〇") || strings.HasPrefix(a, "０") {
						rd, err = rules.ReadGenericSeries(parts[:1], last)
					} else {
						rd, err = rules.ReadPointHyphen(parts[:1], last)
					}
					return rd, end, err == nil
				}
			}
		}
		if conn == "・" && (len(parts) > 2 || strings.Contains(a, "．") || numeral.LeadingZeroSmallUnitCoefficient(a) || !pointFraction(b, rules)) {
			// 番地 permits no following counter and excludes each literal
			// ０/〇 prefix. Otherwise the generic stream retains written p.
			address := counter == "" && !numeralPrefix(words, begin, rules.PointDecimalPrefixes)
			for _, p := range parts {
				address = address && !strings.HasPrefix(p, "０") && !strings.HasPrefix(p, "〇") && !strings.Contains(p, "．")
			}
			if address {
				r, err := rules.ReadAddress(parts[:len(parts)-1], last)
				return r, end, err == nil
			}
			last.NoSpecial = true
			r, err := rules.ReadSeries(parts[:len(parts)-1], last)
			return r, end, err == nil
		}
		if conn == "点" {
			eng := false
			for _, p := range parts {
				for _, ch := range p {
					eng = eng || rules.Digits[string(ch)].Type == "Eng"
				}
			}
			if eng || mixed || len(parts) > 2 {
				r, err := rules.ReadConnectorPoint(parts, last)
				return r, end, err == nil
			}
		}
		last.Number = strings.Join(parts, "．")
		r, err := rules.Read(last)
		return r, end, err == nil
	}
	if conn == "（" || conn == "）" {
		// Native consumes either bracket between numbers into the same
		// numeral, retaining a written pause and the final counter table
		// reading. Bracket runs do not use whole-number counter specials.
		last.NoSpecial = true
		if counter != "" {
			end = e
		}
		if bracketPhone(parts) || (counter == "" || counter == "番" || counter == "号") && phonePrefix(words, begin) {
			r, err := rules.ReadTelephone(parts, last)
			return r, end, err == nil
		}
		r, err := rules.ReadSeries(parts[:len(parts)-1], last)
		return r, end, err == nil
	}
	// Native numeric connector patterns distinguish Jap and Eng digit runs.
	// A mixed run uses written pauses and suppresses whole-counter specials.
	if (conn == "−" || conn == "ー" || conn == "―" || conn == "／" || conn == "：") && mixedNumberScripts(parts, rules) {
		last.NoSpecial = true
		if counter != "" {
			end = e
		}
		r, err := rules.ReadSeries(parts[:len(parts)-1], last)
		return r, end, err == nil
	}
	switch conn {
	case "～":
		if len(parts) != 2 {
			return nil, end, false
		}
		if counter != "" {
			end = e
		}
		r, err := rules.ReadRange(a, "カラ", last)
		return r, end, err == nil
	case "対":
		if len(parts) != 2 {
			return nil, end, false
		}
		if counter != "" {
			end = e
		}
		r, err := rules.ReadRange(a, "タイ", last)
		return r, end, err == nil
	case "／":
		if len(parts) == 2 && utf8.RuneCountInString(a) == 4 && f.validDate(parts) {
			r, err := rules.ReadDate(parts...)
			return r, end, err == nil
		}
		if len(parts) == 3 && utf8.RuneCountInString(a) == 4 && f.validDate(parts) {
			// the day takes no counter of its own here
			r, err := rules.ReadDate(parts...)
			return r, end, err == nil
		}
		if len(parts) == 3 {
			r, err := rules.ReadSeries(parts[:2], last)
			return r, end, err == nil
		}
		if len(parts) != 2 {
			return nil, end, false
		}
		r, err := rules.ReadFraction(a, b)
		return r, end, err == nil
	case "：":
		if f.isTime(parts) {
			r, err := rules.ReadTime(parts...)
			return r, end, err == nil
		}
		if counter != "" {
			r, err := rules.ReadLiteralCounterSeries(parts[:len(parts)-1], last)
			return r, e, err == nil
		}
		r, err := rules.ReadRatio(parts[:len(parts)-1], numeral.Request{Number: b})
		return r, end, err == nil
	case "−", "ー", "―":
		if digitParts(parts) && counter == "" {
			switch {
			case isPhone(parts):
				r, err := rules.ReadDigits(parts, true)
				return r, end, err == nil
			case readDigitwise(parts):
				r, err := rules.ReadDigits(parts, false)
				return r, end, err == nil
			}
		}
		zeroPrefix := func(s string) bool { return strings.HasPrefix(s, "０") || strings.HasPrefix(s, "〇") }
		if len(parts) == 2 && !strings.Contains(a, "．") && !strings.Contains(b, "．") && !rules.ZeroRangeCounters[counter] && (zeroPrefix(a) || zeroPrefix(b)) {
			if counter == "" && (a == "０" || a == "〇" || b == "０" || b == "〇") && utf8.RuneCountInString(a) <= 3 && utf8.RuneCountInString(b) <= 3 && digitPartsNormalized(parts, rules) {
				rd, err := rules.ReadRange(a, "タイ", last)
				return rd, end, err == nil
			}
			last.NoSpecial = true
			if counter != "" {
				end = e
			}
			rd, err := rules.ReadGenericSeries(parts[:len(parts)-1], last)
			return rd, end, err == nil
		}
		if counter == "" && len(parts) == 2 && f.particleFollows(words, end) {
			r, err := rules.ReadRange(a, "タイ", numeral.Request{Number: b})
			return r, end, err == nil
		}
		if len(parts) != 2 {
			if counter != "" {
				return nil, end, false
			}
			r, err := rules.ReadHyphen(parts[:len(parts)-1], last)
			return r, end, err == nil
		}
		if counter != "" || strings.Contains(a, "．") || strings.Contains(b, "．") {
			if counter != "" {
				end = e
			}
			r, err := rules.ReadRange(a, "カラ", last)
			return r, end, err == nil
		}
		r, err := rules.ReadHyphen(parts[:1], numeral.Request{Number: b})
		return r, end, err == nil
	}
	return nil, end, false
}

// These are the pre-numeral contexts of the native 電話 pattern. Spaces
// before the number do not prevent the context from matching.
func phonePrefix(words []frontend.Word, begin int) bool {
	var before string
	for i := begin - 1; i >= 0 && utf8.RuneCountInString(before) < 10; i-- {
		before = words[i].Text + before
	}
	before = strings.TrimSpace(before)
	for _, prefix := range []string{"電話番号", "ＦＡＸ番号", "電話", "ＦＡＸ", "ＦＡＸ：", "ＴＥＬ", "ＴＥＬ：", "電話ＴＥＬ"} {
		if strings.HasSuffix(before, prefix) {
			return true
		}
	}
	return false
}

func numeralPrefix(words []frontend.Word, begin int, prefixes map[string]bool) bool {
	var before string
	for i := begin - 1; i >= 0 && utf8.RuneCountInString(before) < 10; i-- {
		before = words[i].Text + before
	}
	before = strings.TrimSpace(before)
	for prefix := range prefixes {
		if strings.HasSuffix(before, prefix) {
			return true
		}
	}
	return false
}

func bracketPhone(parts []string) bool {
	if len(parts) != 3 {
		return false
	}
	zero := strings.HasPrefix(parts[0], "０") || strings.HasPrefix(parts[0], "〇")
	return zero && utf8.RuneCountInString(parts[0]) <= 3 && utf8.RuneCountInString(parts[1]) <= 4 && utf8.RuneCountInString(parts[2]) == 4
}

func mixedNumberScripts(parts []string, rules *numeral.Rules) bool {
	kind := ""
	for _, p := range parts {
		for _, c := range p {
			next := rules.Digits[string(c)].Type
			if next != "Jap" && next != "Eng" {
				continue
			}
			if kind != "" && kind != next {
				return true
			}
			kind = next
		}
	}
	return false
}

// isTime reports whether the numbers read as H：MM(：SS).
func (f *Frontend) isTime(parts []string) bool {
	if len(parts) < 2 || len(parts) > 3 || utf8.RuneCountInString(parts[0]) > 2 {
		return false
	}
	for _, p := range parts[1:] {
		if utf8.RuneCountInString(p) != 2 {
			return false
		}
	}
	return true
}

// validDate checks Y／M(／D): a four digit year starting with 1 or 2, a month
// of 1 to 12 and a day of 1 to 31.
func (f *Frontend) validDate(parts []string) bool {
	vals := make([]int, len(parts))
	for i, p := range parts {
		for _, c := range p {
			if c < '０' || c > '９' {
				return false
			}
			vals[i] = vals[i]*10 + int(c-'０')
		}
	}
	if y := vals[0]; y < 1000 || y > 2999 {
		return false
	}
	if vals[1] < 1 || vals[1] > 12 {
		return false
	}
	return len(parts) < 3 || vals[2] >= 1 && vals[2] <= 31
}

// accentText formats groups the way the dictionary and the numeral pass store
// accents: "rise-fall-moras" separated by commas, "s0" for a pause.
func accentText(gs []numeral.Group) string {
	parts := make([]string, len(gs))
	for i, g := range gs {
		if g.Pause {
			parts[i] = "s0"
			if g.Strong {
				parts[i] = "s1"
			}
			continue
		}
		parts[i] = strconv.Itoa(g.Up) + "-" + strconv.Itoa(g.Down) + "-" + strconv.Itoa(g.Moras)
	}
	return strings.Join(parts, ",")
}

func joinGroups(gs []numeral.Group) (read, sound string) {
	for _, g := range gs {
		read += g.Read
		if !g.Pause {
			sound += g.Sound
		}
	}
	return
}

// splitGroups cuts a reading's groups into pieces of two accent groups; a pause
// that directly follows the second group stays with it.
func splitGroups(gs []numeral.Group) [][]numeral.Group {
	var out [][]numeral.Group
	for len(gs) > 0 {
		real, n := 0, 0
		for n < len(gs) && real < 2 {
			if !gs[n].Pause {
				real++
			}
			n++
		}
		if real == 2 && n < len(gs) && gs[n].Pause {
			n++
		}
		out = append(out, gs[:n])
		gs = gs[n:]
	}
	return out
}

// numeralMorph builds the merged word and its detached chain: the first two
// accent groups stay in the word, the rest go to chain words of two groups each
// (native detached expansion chain, +0x28/+0x30).
func numeralMorph(surface string, readings []numeral.Reading) (*Morph, []*Morph) {
	const numeralPOS = 0x1060000
	m := &Morph{Surface: surface, Base: surface, POS: numeralPOS, Moras: -1, Up: -1, Down: -1}
	var chain []*Morph
	for ri, rd := range readings {
		pieces := splitGroups(rd.Groups)
		read, sound := joinGroups(pieces[0])
		if rd.Name != "" {
			read = rd.Name
		}
		connection := rd.Connection
		if connection == "" {
			connection = "*"
		}
		m.Cands = append(m.Cands, Candidate{Reading: read, Pronunciation: sound, Accent: accentText(pieces[0]), Connection: connection})
		if ri == 0 {
			for _, piece := range pieces[1:] {
				r, s := joinGroups(piece)
				chain = append(chain, &Morph{POS: numeralPOS, Moras: -1, Up: -1, Down: -1,
					Cands: []Candidate{{Reading: r, Pronunciation: s, Accent: accentText(piece), Connection: "*"}}})
			}
		}
	}
	return m, chain
}

func digitParts(parts []string) bool {
	for _, p := range parts {
		if !digitsOnly(p) {
			return false
		}
	}
	return true
}

// isPhone matches the engine's phone and postal number patterns: a leading 0
// block, the middle block and a four digit last block (0120 123 456 for the
// toll-free form, 123 4567 for postal codes).
func isPhone(parts []string) bool {
	n := func(i int) int { return utf8.RuneCountInString(parts[i]) }
	switch len(parts) {
	case 2:
		return n(0) == 3 && n(1) == 4
	case 3:
		if strings.HasPrefix(parts[0], "０") && n(0) >= 2 && n(0) <= 4 && n(1) >= 1 && n(1) <= 4 && n(2) == 4 {
			return true
		}
		return strings.HasPrefix(parts[0], "０") && n(0) == 4 && n(1) == 3 && n(2) == 3
	}
	return false
}

// readDigitwise reports numbers joined by hyphens that the engine reads digit
// by digit because a block starts with 0 (2024-04-05).
func readDigitwise(parts []string) bool {
	for _, p := range parts {
		if strings.HasPrefix(p, "０") && utf8.RuneCountInString(p) >= 2 {
			return true
		}
	}
	return false
}

// englishPrefixes are the words after which a number from 1 to 10 is read in
// English (ナンバーファイブ).
var englishPrefixes = map[string]bool{"Ｖ": true, "ワースト": true, "ナンバー": true, "ベスト": true, "トップ": true,
	"フラット": true, "Ｎｏ．": true, "№": true}

// particleFollows reports whether one of から, で, に, と, の follows: a score
// such as 7-8に is read as 7対8.
func (f *Frontend) particleFollows(words []frontend.Word, end int) bool {
	if end >= len(words) {
		return false
	}
	switch words[end].Text {
	case "から", "で", "に", "と", "の":
		return true
	}
	return false
}

func digitPartsNormalized(parts []string, rules *numeral.Rules) bool {
	for _, part := range parts {
		for _, ch := range part {
			d, ok := rules.Digits[string(ch)]
			if !ok || d.Type != "Eng" && d.Type != "Jap" {
				return false
			}
		}
	}
	return !mixedNumberScripts(parts, rules)
}

func streamConnector(s string) bool {
	return s == "・" || s == "／" || s == "−" || s == "ー" || s == "―" || s == "（" || s == "）" || s == "："
}
func hyphenConnector(s string) bool { return s == "−" || s == "ー" || s == "―" }
func startsWithUnit(s string, rules *numeral.Rules) bool {
	chars := []rune(s)
	if len(chars) == 0 {
		return false
	}
	_, ok := rules.Units[string(chars[0])]
	return ok
}

func (f *Frontend) patternStream(separators, parts []string, words []frontend.Word, begin, end int, starts map[int]bool) ([]numeral.Reading, int, bool) {
	conn := separators[0]
	// The explicit telephone prefix interprets embedded decimal points
	// as further number blocks before the ordinary decimal-range pattern.
	dottedPhone := phonePrefix(words, begin)
	for _, separator := range separators {
		dottedPhone = dottedPhone && (separator == "・" || hyphenConnector(separator))
	}
	if dottedPhone {
		var blocks []string
		dotted := false
		for _, part := range parts {
			dotted = dotted || strings.Contains(part, "．")
			blocks = append(blocks, strings.Split(part, "．")...)
		}
		validBlocks := true
		for _, block := range blocks {
			validBlocks = validBlocks && block != ""
			for _, ch := range block {
				_, digit := f.Numerals.Digits[string(ch)]
				_, unit := f.Numerals.Units[string(ch)]
				validBlocks = validBlocks && (digit || unit)
			}
		}
		if dotted && validBlocks && !startsWithUnit(blocks[0], f.Numerals) {
			counter, fall, moras, n := f.counterAfter(words, end, starts)
			if counter == "" || counter == "番" || counter == "号" {
				last := numeral.Request{Number: blocks[len(blocks)-1], Counter: counter, CounterFall: fall, CounterMoras: moras, CounterHasInfo: counter != ""}
				rd, err := f.Numerals.ReadPrefixTelephone(blocks, last)
				return rd, n, err == nil
			}
		}
	}

	// A literal 秒 followed by another number belongs to the generic
	// hyphen stream, rather than ending a range at the seconds counter.
	genericSeconds := true
	for _, separator := range separators {
		genericSeconds = genericSeconds && hyphenConnector(separator)
	}
	if genericSeconds {
		counter, fall, moras, e := f.counterAfter(words, end, starts)
		if counter == "秒" && e < len(words) && f.isNumeralWord(words[e]) {
			n, number := f.scanNumber(words, e, starts, true)
			last := numeral.Request{Number: number, NoSpecial: true}
			c, cf, cm, ce := f.counterAfter(words, n, starts)
			if c != "" {
				last.Counter, last.CounterFall, last.CounterMoras, last.CounterHasInfo = c, cf, cm, true
				n = ce
			}
			first := numeral.Request{Number: parts[len(parts)-1], Counter: counter, CounterFall: fall, CounterMoras: moras, CounterHasInfo: true}
			rd, err := f.Numerals.ReadConnectorSeconds(parts[:len(parts)-1], first, last)
			return rd, n, err == nil
		}
	}
	uniform := true
	for _, s := range separators {
		uniform = uniform && (s == conn || (conn == "（" || conn == "）") && (s == "（" || s == "）"))
	}
	if uniform {
		return f.pattern(conn, parts, words, begin, end, starts)
	}
	rules := f.Numerals
	last := numeral.Request{Number: parts[len(parts)-1], NoSpecial: true}
	counter, fall, moras, e := f.counterAfter(words, end, starts)
	if counter != "" {
		last.Counter = counter
		last.CounterFall = fall
		last.CounterMoras = moras
		last.CounterHasInfo = true
		end = e
	}
	telephone := phonePrefix(words, begin) && separators[0] != "）" && !startsWithUnit(parts[0], rules) && (counter == "" || counter == "番" || counter == "号")
	brackets := false
	for _, s := range separators {
		brackets = brackets || s == "（" || s == "）"
		telephone = telephone && (s == "・" || hyphenConnector(s) || s == "（" || s == "）")
	}
	if telephone {
		rd, err := rules.ReadPrefixTelephone(parts, last)
		return rd, end, err == nil
	}
	if brackets {
		rd, err := rules.ReadSeries(parts[:len(parts)-1], last)
		return rd, end, err == nil
	}
	if !mixedNumberScripts(parts, rules) && len(separators) == 2 {
		if separators[0] == "：" && separators[1] == "・" && !startsWithUnit(parts[2], rules) {
			last.Number = parts[1] + "．" + parts[2]
			rd, err := rules.ReadRatio(parts[:1], last)
			return rd, end, err == nil
		}
		if separators[0] == "・" && hyphenConnector(separators[1]) && !startsWithUnit(parts[1], rules) {
			rd, err := rules.ReadRange(parts[0]+"．"+parts[1], "カラ", last)
			return rd, end, err == nil
		}
		if hyphenConnector(separators[0]) && separators[1] == "・" && !startsWithUnit(parts[2], rules) {
			last.Number = parts[1] + "．" + parts[2]
			rd, err := rules.ReadRange(parts[0], "カラ", last)
			return rd, end, err == nil
		}
	}
	rd, err := rules.ReadGenericSeries(parts[:len(parts)-1], last)
	return rd, end, err == nil
}

// The small-number decimal grammar accepts digits, optionally followed by
// one unit. Interior units or another point fall through to address/series.
func pointFraction(s string, rules *numeral.Rules) bool {
	chars := []rune(s)
	if len(chars) > 0 && strings.ContainsRune("十百千万億兆京垓", chars[len(chars)-1]) {
		chars = chars[:len(chars)-1]
	}
	if len(chars) == 0 {
		return false
	}
	for _, c := range chars {
		d, ok := rules.Digits[string(c)]
		if !ok || d.Type != "Eng" && d.Type != "Jap" {
			return false
		}
	}
	return true
}

// Upper-case J in a label pattern includes written Japanese units.
func japaneseNumberParts(parts []string, rules *numeral.Rules) bool {
	for _, part := range parts {
		for _, c := range part {
			if d, ok := rules.Digits[string(c)]; ok && d.Type == "Jap" {
				continue
			}
			if u, ok := rules.Units[string(c)]; ok && u.Type == "Unit" {
				continue
			}
			return false
		}
	}
	return true
}
