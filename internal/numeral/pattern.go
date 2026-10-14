package numeral

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// plain reads a number without counter and special readings.
func (r *Rules) plainChunks(number string) ([]chunk, error) {
	v, err := Value(number)
	if err != nil {
		return nil, err
	}
	return r.numberChunks(v), nil
}

// ReadRange reads "A～B" (con "カラ") and "A対B" (con "タイ"): the first number
// takes the connector as a counter-like word, the second is read as usual,
// with its own counter when it has one.
func (r *Rules) ReadRange(a, con string, b Request) ([]Reading, error) {
	if b.Counter != "" {
		b.NoSpecial = true // 1-3日 reads サンニチ, not ミッカ
		// A decimal on the left retains the ordinary fractional counter
		// mode (10f0a0 param11=0); an integer prefix rebuilds numeric keys.
		b.rangeTail = !(con == "カラ" && strings.Contains(a, "．"))
		b.genericDecimal = con == "タイ"
	}
	if con == "カラ" && strings.Contains(a, "．") {
		// a decimal keeps its groups; the last one takes カラ without changing its accent
		rs, err := r.decimal(Request{Number: a})
		if err != nil {
			return nil, err
		}
		groups := rs[0].Groups
		g := &groups[len(groups)-1]
		g.Read, g.Sound, g.Moras = g.Read+"カラ", g.Sound+"カラ", g.Moras+2
		rb, err := r.Read(b)
		if err != nil {
			return nil, err
		}
		return []Reading{{Groups: append(groups, rb[0].Groups...)}}, nil
	}
	prefix, err := r.connectorFragment(Request{Number: a, NoSpecial: true})
	if err != nil {
		return nil, err
	}
	groups := prefix[0].Groups
	last := &groups[len(groups)-1]
	switch con {
	case "カラ":
		last.Read += "カラ"
		last.Sound += "カラ"
		last.Moras += 2
	case "タイ":
		tail := *last
		cs, err := r.numericSoundChunks(tail.Sound)
		if err != nil {
			cs = []chunk{{Read: tail.Read, Sound: tail.Sound}}
		}
		rd, err := r.withCounter(Request{Number: a, Counter: "対比", CounterFall: 1, CounterMoras: 2}, cs, false)
		if err != nil {
			return nil, err
		}
		groups = append(groups[:len(groups)-1], rd[0].Groups...)
	default:
		return nil, fmt.Errorf("unsupported connector %q", con)
	}
	rb, err := r.Read(b)
	if err != nil {
		return nil, err
	}
	return []Reading{{Groups: append(groups, rb[0].Groups...)}}, nil
}

// ReadFraction reads "numerator／denominator" as "denominator ブンノ
// numerator" (１／２ ニブンノイチ).
func (r *Rules) ReadFraction(numerator, denominator string) ([]Reading, error) {
	chunks, err := r.plainChunks(denominator)
	if err != nil {
		return nil, err
	}
	var groups []Group
	for _, c := range chunks[:len(chunks)-1] {
		g := r.accentOf(c)
		r.flatBeforeMore(c, &g)
		groups = append(groups, g)
	}
	last := r.accentOf(chunks[len(chunks)-1])
	groups = append(groups, Group{Read: last.Read + "ブンノ", Sound: last.Sound + "ブンノ", Up: 2, Down: 0, Moras: last.Moras + 3})
	rn, err := r.Read(Request{Number: numerator, NoSpecial: true})
	if err != nil {
		return nil, err
	}
	return []Reading{{Groups: append(groups, rn[0].Groups...)}}, nil
}

// ReadTime reads "H：M" and "H：M：S" (digits as written, so 05 and 00 are
// minutes): hour with 時, minutes with 分 and seconds with 秒, with a short
// pause after the hour and after the minutes when seconds follow.
func (r *Rules) ReadTime(parts ...string) ([]Reading, error) {
	if len(parts) < 2 || len(parts) > 3 {
		return nil, fmt.Errorf("unsupported time")
	}
	units := []string{"時", "分", "秒"}
	var groups []Group
	for i, p := range parts {
		if i > 0 {
			groups = append(groups, Group{Pause: true, Silent: true})
		}
		v, err := Value(p)
		if err != nil {
			return nil, err
		}
		if v.Sign() == 0 {
			switch units[i] {
			case "時":
				groups = append(groups, group("レイジ", "レージ", 2, 3))
			case "分":
				groups = append(groups, group("レイフン", "レーフン", 2, 4))
			default:
				groups = append(groups, group("レイビョウ", "レービョー", 2, 4))
			}
			continue
		}
		ctr := r.Counters[units[i]]
		rs, err := r.Read(Request{Number: p, Counter: units[i], NoSpecial: true, CounterFall: 1, CounterMoras: MoraCount(ctr.Sound)})
		if err != nil {
			return nil, err
		}
		gs := rs[0].Groups
		if units[i] == "分" {
			// the reading keeps フン while the pronunciation is プン (ジュップン)
			gs[len(gs)-1].Read = strings.Replace(gs[len(gs)-1].Read, "プン", "フン", 1)
		}
		groups = append(groups, gs...)
	}
	return []Reading{{Groups: groups}}, nil
}

// ReadHyphen reads numbers joined by hyphens as "A ノ B ノ C": every number
// but the last has its last group take ノ (keeping the accent), followed by a
// silent pause.
func (r *Rules) ReadHyphen(parts []string, last Request) ([]Reading, error) {
	return r.readHyphen(parts, last, false)
}
func (r *Rules) ReadPointHyphen(parts []string, last Request) ([]Reading, error) {
	return r.readHyphen(parts, last, true)
}
func (r *Rules) readHyphen(parts []string, last Request, point bool) ([]Reading, error) {
	var groups []Group
	for _, a := range parts {
		readings, err := r.connectorFragment(Request{Number: a, NoSpecial: true})
		if err != nil {
			return nil, err
		}
		prefix := readings[0].Groups
		g := &prefix[len(prefix)-1]
		g.Read += "ノ"
		g.Sound += "ノ"
		g.Moras++
		groups = append(groups, prefix...)
		groups = append(groups, Group{Pause: true, Silent: true})
	}
	read := r.Read
	if point && writtenLargeUnits(last.Number) {
		read = r.connectorUnitWords
	}
	rb, err := read(last)
	if err != nil {
		return nil, err
	}
	return []Reading{{Groups: append(groups, rb[0].Groups...)}}, nil
}

// ReadSeconds reads a literal 秒 between numeric fragments. The 時間
// pattern reads its tail digitwise, like a fractional part; the fallback
// 100a90 path reads both fragments normally with an s0 boundary.
func (r *Rules) secondsPrefix(first Request, fractional bool) ([]Group, error) {
	first.NoSpecial = true
	writtenUnit := false
	for _, unit := range bigUnits {
		writtenUnit = writtenUnit || strings.HasSuffix(first.Number, string(unit))
	}
	anyUnit := strings.ContainsAny(first.Number, "十百千"+string(bigUnits))
	if fractional && leadingZeroDigitRun(first.Number) {
		// The time pattern evaluates its integer before attaching 秒.
		// Unlike the generic mixed-script stream, all-zero integers become
		// the single zero used by the fractional レービョー reading.
		zero := string([]rune(first.Number)[0])
		first.Number = strings.TrimLeft(first.Number, zero)
		if first.Number == "" {
			first.Number = zero
		}
	}
	if anyUnit && fractional {
		// The zero-skipping pattern precedes the seconds-fraction pattern.
		// Its literal zero may be followed by a written unit (０京秒１).
		if strings.HasPrefix(first.Number, "０") || strings.HasPrefix(first.Number, "〇") {
			zero := string([]rune(first.Number)[0])
			first.Number = strings.TrimLeft(first.Number, zero)
		}
	}
	var prefix []Group
	if fractional && (first.Number == "０" || first.Number == "〇" || first.Number == "零") {
		prefix = []Group{group("レイビョウ", "レービョー", 2, 4)}
	} else {
		plain := first
		if anyUnit {
			plain.Counter = ""
		}
		// A mixed-script 秒 stream is handled inside 100a90, without
		// matching the whole-number zero-skipping counter pattern.
		rd, err := r.read(plain, !fractional && !anyUnit)
		if err != nil {
			return nil, err
		}
		prefix = rd[0].Groups
		if anyUnit {
			second := r.Cons["秒"]
			w := r.BySound[second.Sound]
			g := &prefix[len(prefix)-1]
			bareWord := false
			for _, u := range r.Units {
				bareWord = bareWord || u.Type == "Unit" && g.Sound == u.Sound
			}
			if len(prefix) > 1 && prefix[len(prefix)-2].Pause {
				// A paired-digit boundary (零ｂ京) keeps the following
				// unit and literal 秒 as independent table words.
				bareWord = false
			}
			if bareWord {
				fall := g.Down
				if fall == 0 {
					fall = g.Moras
				}
				*g = group(g.Read+second.Read, g.Sound+second.Sound, fall, g.Moras+w.Moras)
			} else if writtenUnit {
				prefix = append(prefix, group(second.Read, second.Sound, w.Fall, w.Moras))
			} else {
				rd, err = r.Read(first)
				if err != nil {
					return nil, err
				}
				prefix = rd[0].Groups
			}
		}
	}
	return prefix, nil
}

func (r *Rules) ReadSeconds(first, last Request, fractional bool) ([]Reading, error) {
	last.NoSpecial = true
	prefix, err := r.secondsPrefix(first, fractional)
	if err != nil {
		return nil, err
	}
	if fractional {
		tail, err := r.decimalFraction(last, last.Number)
		if err != nil {
			return nil, err
		}
		return []Reading{{Groups: append(prefix, tail...)}}, nil
	}
	tail, err := r.Read(last)
	if err != nil {
		return nil, err
	}
	prefix = append(prefix, Group{Pause: true, Silent: true})
	return []Reading{{Groups: append(prefix, tail[0].Groups...)}}, nil
}

// ReadConnectorSeconds keeps literal 秒 inside the connector's generic
// numeric stream. Only the final counter applies to the complete stream.
func (r *Rules) ReadConnectorSeconds(parts []string, first, last Request) ([]Reading, error) {
	var groups []Group
	for _, part := range parts {
		rd, err := r.connectorFragment(Request{Number: part, NoSpecial: true})
		if err != nil {
			return nil, err
		}
		groups = append(groups, rd[0].Groups...)
		groups = append(groups, Group{Read: "ｐ", Pause: true, Strong: true})
	}
	prefix, err := r.secondsPrefix(first, false)
	if err != nil {
		return nil, err
	}
	tail, err := r.ReadGenericSeries(nil, last)
	// A multi-digit tail retains 10b010's final numeric counter key;
	// a single digit after literal 秒 instead uses the generic word key.
	digitTail := utf8.RuneCountInString(last.Number) > 1
	for _, ch := range last.Number {
		_, digit := r.Digits[string(ch)]
		digitTail = digitTail && digit
	}
	if digitTail {
		tail, err = r.connectorFragment(last)
	}
	if err != nil {
		return nil, err
	}
	groups = append(groups, prefix...)
	groups = append(groups, Group{Pause: true, Silent: true})
	return []Reading{{Groups: append(groups, tail[0].Groups...)}}, nil
}

// ReadAddress reads the native 番地 stream: whole number fragments followed
// by ノ and an s0 boundary, preserving literal units and paired 零 groups.
func (r *Rules) ReadAddress(parts []string, last Request) ([]Reading, error) {
	var groups []Group
	for _, number := range parts {
		rd, err := r.Read(Request{Number: number, NoSpecial: true})
		if err != nil {
			return nil, err
		}
		gs := rd[0].Groups
		g := &gs[len(gs)-1]
		g.Read, g.Sound, g.Moras = g.Read+"ノ", g.Sound+"ノ", g.Moras+1
		groups = append(groups, gs...)
		groups = append(groups, Group{Pause: true, Silent: true})
	}
	last.NoSpecial = true
	tail, err := r.Read(last)
	if err != nil {
		return nil, err
	}
	return []Reading{{Groups: append(groups, tail[0].Groups...)}}, nil
}

// ReadSeries reads separated numbers that are not a date as plain
// numbers with a written pause (ｐ) after each but the last (3/4/5).
func (r *Rules) ReadSeries(parts []string, last Request) ([]Reading, error) {
	return r.readSeries(parts, last, false)
}

func (r *Rules) readSeries(parts []string, last Request, genericPoints bool) ([]Reading, error) {
	var groups []Group
	for _, a := range parts {
		rs, err := r.Read(Request{Number: a, NoSpecial: true, genericDecimal: genericPoints, connectorStream: true})
		if err != nil {
			return nil, err
		}
		if leadingZeroDigitRun(a) {
			chars := []rune(a)
			if len(chars)%2 == 1 && (kanjiDigits[chars[len(chars)-1]] == 2 || kanjiDigits[chars[len(chars)-1]] == 5) {
				// The paired 1043f0 prefix keeps its lone 2/5 long before p.
				w := fractionPair[digitRune(chars[len(chars)-1])]
				rs[0].Groups[len(rs[0].Groups)-1] = group(w.Read, w.Sound, 1, MoraCount(w.Sound))
			}
		}
		groups = append(groups, rs[0].Groups...)
		groups = append(groups, Group{Read: "ｐ", Pause: true, Strong: true})
	}
	rb, err := r.writtenPauseFragment(last)
	if err != nil {
		return nil, err
	}
	return []Reading{{Groups: append(groups, rb[0].Groups...)}}, nil
}

// ReadLiteralCounterSeries selects the generic point reader for a colon
// expression with a trailing counter; ordinary ratios have no such counter.
func (r *Rules) ReadLiteralCounterSeries(parts []string, last Request) ([]Reading, error) {
	last.NoSpecial, last.genericDecimal = true, true
	return r.readSeries(parts, last, true)
}

// ReadTelephone reads digit pairs separated by ノ. A final counter attaches
// to the final pair without applying single-digit sound changes or specials.
func (r *Rules) ReadTelephone(blocks []string, last Request) ([]Reading, error) {
	readings, err := r.ReadDigits(blocks, true)
	if err != nil || last.Counter == "" {
		return readings, err
	}
	groups := readings[0].Groups
	tail := groups[len(groups)-1]
	merged, err := r.withCounter(last, []chunk{{Read: tail.Read, Sound: tail.Sound}}, false)
	if err != nil {
		return nil, err
	}
	readings[0].Groups = append(groups[:len(groups)-1], merged[0].Groups...)
	return readings, nil
}

// The explicit phone-prefix pattern lengthens an isolated 2/5 before ノ
// when another block follows; the final digit keeps its short reading.
func (r *Rules) ReadPrefixTelephone(parts []string, last Request) ([]Reading, error) {
	if len(parts) == 1 && strings.ContainsAny(last.Number, "十百千"+string(bigUnits)) {
		return r.connectorFragment(last)
	}
	// A bare written unit between phone blocks retains its own table
	// accent; the following ノ is an independent connector word.
	for i, part := range parts[:len(parts)-1] {
		if u, ok := r.Units[part]; ok && u.Type == "Unit" {
			head, err := r.ReadPrefixTelephone(parts[:i+1], Request{Number: part})
			if err != nil {
				return nil, err
			}
			tail, err := r.ReadPrefixTelephone(parts[i+1:], last)
			if err != nil {
				return nil, err
			}
			no := r.BySound["ノ"]
			connector := group(no.Read, no.Sound, no.Fall, no.Moras)
			connector.Up = 0
			groups := append(head[0].Groups, connector)
			groups = append(groups, Group{Read: "ｐ", Pause: true, Strong: true})
			return []Reading{{Groups: append(groups, tail[0].Groups...)}}, nil
		}
	}
	if _, unit := r.Units[last.Number]; unit || strings.ContainsAny(last.Number, "十百千"+string(bigUnits)) {
		rd, err := r.ReadDigits(parts[:len(parts)-1], true)
		if err != nil {
			return nil, err
		}
		gs := rd[0].Groups
		g := &gs[len(gs)-1]
		g.Read += "ノ"
		g.Sound += "ノ"
		g.Moras++
		if g.Sound == "ニノ" || g.Sound == "ゴノ" {
			g.Read = strings.Replace(g.Read, "ノ", "ーノ", 1)
			g.Sound = strings.Replace(g.Sound, "ノ", "ーノ", 1)
			g.Moras++
		}
		tail, err := r.connectorFragment(last)
		if err != nil {
			return nil, err
		}
		gs = append(gs, Group{Read: "ｐ", Pause: true, Strong: true})
		return []Reading{{Groups: append(gs, tail[0].Groups...)}}, nil
	}
	rd, err := r.ReadTelephone(parts, last)
	if err != nil {
		return nil, err
	}
	for i := range rd[0].Groups {
		g := &rd[0].Groups[i]
		if g.Sound == "ニノ" || g.Sound == "ゴノ" {
			g.Read, g.Sound = strings.Replace(g.Read, "ノ", "ーノ", 1), strings.Replace(g.Sound, "ノ", "ーノ", 1)
			g.Moras++
		}
	}
	return rd, nil
}

// ReadRatio reads "A：B：C" as "A タイ B タイ C".
func (r *Rules) ReadRatio(parts []string, last Request) ([]Reading, error) {
	if len(parts) < 1 {
		return nil, fmt.Errorf("a ratio needs two numbers")
	}
	var groups []Group
	for _, p := range parts {
		chunks, err := r.plainChunks(p)
		if err != nil {
			return nil, err
		}
		rs, err := r.withCounter(Request{Number: p, Counter: "対比", CounterFall: 1, CounterMoras: 2}, chunks, false)
		if err != nil {
			return nil, err
		}
		groups = append(groups, rs[0].Groups...)
	}
	rb, err := r.Read(last)
	if err != nil {
		return nil, err
	}
	return []Reading{{Groups: append(groups, rb[0].Groups...)}}, nil
}

// ReadDate reads "Y／M／D" (or "Y／M") as year, month and day with a short
// pause after the year and after the month.
func (r *Rules) ReadDate(parts ...string) ([]Reading, error) {
	units := []string{"年", "月", "日"}
	if len(parts) < 2 || len(parts) > 3 {
		return nil, fmt.Errorf("unsupported date")
	}
	var groups []Group
	for i, p := range parts {
		ctr := r.Counters[units[i]]
		rs, err := r.Read(Request{Number: p, Counter: units[i], CounterFall: 1, CounterMoras: MoraCount(ctr.Sound), CounterHasInfo: true})
		if err != nil {
			return nil, err
		}
		groups = append(groups, rs[0].Groups...)
		if i < len(parts)-1 {
			groups = append(groups, Group{Read: "ｂ", Pause: true})
		}
	}
	return []Reading{{Groups: groups}}, nil
}

// ReadDigits reads blocks of digits joined by separators one digit at a time,
// two digits to an accent group. In a phone number (phone) the last group
// before a separator takes ノ and the pause after it is not written; otherwise
// the pause between blocks is written as ｐ.
func (r *Rules) ReadDigits(blocks []string, phone bool) ([]Reading, error) {
	var groups []Group
	for bi, block := range blocks {
		var digits []rune
		for _, c := range block {
			if _, ok := kanjiDigits[c]; !ok {
				return nil, fmt.Errorf("unsupported digit %q", c)
			}
			if c == '零' {
				digits = append(digits, c)
			} else {
				digits = append(digits, digitRune(c))
			}
		}
		last := bi == len(blocks)-1
		word := func(d rune) Word {
			if d == '零' {
				return r.Digits["零"].Word
			}
			if d == '〇' {
				return Word{Read: "ゼロ", Sound: "ゼロ"}
			}
			return fractionPair[d]
		}
		for i := 0; i < len(digits); i += 2 {
			var g Group
			if i+1 < len(digits) {
				a, b := word(digits[i]), word(digits[i+1])
				g = group(a.Read+b.Read, a.Sound+b.Sound, 3, 4)
			} else if !phone && len(blocks) > 1 {
				w := word(digits[i])
				g = group(w.Read, w.Sound, 1, 2)
			} else if digits[i] == '〇' {
				g = group("ゼロ", "ゼロ", 1, 2)
			} else {
				w := r.digitWordByKey(string(digits[i]))
				g = r.accentOf(chunk{Read: w.Read, Sound: w.Sound, Key: string(digits[i]), TkKey: string(digits[i])})
			}
			endOfBlock := i+2 >= len(digits)
			if endOfBlock && !last && phone {
				g = Group{Read: g.Read + "ノ", Sound: g.Sound + "ノ", Up: g.Up, Down: g.Down, Moras: g.Moras + 1}
			}
			groups = append(groups, g)
			switch {
			case !endOfBlock:
				groups = append(groups, Group{Read: "ｂ", Pause: true})
			case !last:
				p := Group{Pause: true, Strong: true}
				if !phone {
					p.Read = "ｐ"
				}
				groups = append(groups, p)
			}
		}
	}
	return []Reading{{Groups: groups}}, nil
}

// ReadEnglish reads 1 to 10 in English (ナンバーファイブ, ベストテン), as the
// engine does after words such as ナンバー and ベスト.
func (r *Rules) ReadEnglish(number string) ([]Reading, error) {
	rs := []rune(number)
	var eng string
	switch {
	case len(rs) == 1 && kanjiDigits[rs[0]] > 0:
		key := digitKanji[kanjiDigits[rs[0]]]
		for _, d := range r.Digits {
			if d.Type == "Eng" && d.Jap == key && d.Eng != "" {
				eng = d.Eng
			}
		}
	case number == "１０":
		return []Reading{{Groups: []Group{group("テン", "テン", 1, 2)}}}, nil
	}
	if eng == "" {
		return nil, fmt.Errorf("%q has no English reading", number)
	}
	if sp, ok := r.Specials[number]; ok && sp.Display == eng {
		return r.special(sp)
	}
	return []Reading{{Groups: []Group{r.accentOf(chunk{Read: eng, Sound: eng})}}}, nil
}

// ReadGenericSeries is 100a90's written-pause fallback. Numeric fragment
// spellings survive; a final digit has no whole-counter sound-change key.
func (r *Rules) ReadGenericSeries(parts []string, last Request) ([]Reading, error) {
	var groups []Group
	unitPrefix := false
	for _, a := range parts {
		unitPrefix = unitPrefix || strings.ContainsAny(a, "十百千"+string(bigUnits))
		rd, err := r.connectorFragment(Request{Number: a, NoSpecial: true})
		if err != nil {
			return nil, err
		}
		groups = append(groups, rd[0].Groups...)
		groups = append(groups, Group{Read: "ｐ", Pause: true, Strong: true})
	}
	counter := last.Counter
	if unitPrefix && counter != "" && !writtenLargeUnits(last.Number) {
		last.NoSpecial = true
		tail, err := r.writtenPauseFragment(last)
		if err != nil {
			return nil, err
		}
		return []Reading{{Groups: append(groups, tail[0].Groups...)}}, nil
	}
	last.Counter = ""
	last.NoSpecial = true
	read := r.connectorFragment
	if len(parts) > 0 {
		read = r.writtenPauseFragment
	}
	// Written large-unit tails keep the first unit's separate accent group.
	if writtenLargeUnits(last.Number) {
		read = r.connectorUnitWords
	}
	rd, err := read(last)
	if err != nil {
		return nil, err
	}
	tail := rd[0].Groups
	if counter != "" {
		last.Counter = counter
		g := tail[len(tail)-1]
		key := ""
		chars := []rune(last.Number)
		if len(chars) > 0 {
			if _, ok := r.Units[string(chars[len(chars)-1])]; ok {
				key = string(chars[len(chars)-1])
			}
		}
		cs := []chunk{{Read: g.Read, Sound: g.Sound, Key: key, TkKey: key, numericFragment: true}}
		with, err := r.withCounter(last, cs, false)
		if err != nil {
			return nil, err
		}
		tail = append(tail[:len(tail)-1], with[0].Groups...)
	}
	return []Reading{{Groups: append(groups, tail...)}}, nil
}
