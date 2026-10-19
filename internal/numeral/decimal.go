package numeral

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// decimalDigits are the readings of a digit inside a decimal fraction when two
// digits form one accent group (the digits 2 and 5 are read long) and for the
// last digit of a number before テン.
var fractionPair = map[rune]Word{
	'〇': {Read: "レイ", Sound: "レー"}, '一': {Read: "イチ", Sound: "イチ"}, '二': {Read: "ニー", Sound: "ニー"},
	'三': {Read: "サン", Sound: "サン"}, '四': {Read: "ヨン", Sound: "ヨン"}, '五': {Read: "ゴー", Sound: "ゴー"},
	'六': {Read: "ロク", Sound: "ロク"}, '七': {Read: "ナナ", Sound: "ナナ"}, '八': {Read: "ハチ", Sound: "ハチ"},
	'九': {Read: "キュウ", Sound: "キュー"},
}

// beforeTen are the digits read before テン (一 and 八 take ッ, 二 and 五 are long).
var beforeTen = map[string]Word{
	"〇": {Read: "レイ", Sound: "レー"}, "一": {Read: "イッ", Sound: "イッ"}, "二": {Read: "ニー", Sound: "ニー"},
	"三": {Read: "サン", Sound: "サン"}, "四": {Read: "ヨン", Sound: "ヨン"}, "五": {Read: "ゴー", Sound: "ゴー"},
	"六": {Read: "ロク", Sound: "ロク"}, "七": {Read: "ナナ", Sound: "ナナ"}, "八": {Read: "ハッ", Sound: "ハッ"},
	"九": {Read: "キュウ", Sound: "キュー"},
}

// decimal reads "integer．fraction" (and a counter after the last fractional
// digit): the integer part's last chunk takes テン, the fraction is read digit
// by digit in pairs with a pause after each pair that is followed by more.
func (r *Rules) decimal(req Request) ([]Reading, error) {
	if req.genericDecimal {
		return r.connectorDecimal(req)
	}
	// The fractional patterns accept digits, optionally followed by one
	// written unit. Other unit streams remain on 100a90's connector path.
	partsForUnits := strings.Split(req.Number, "．")
	if partsForUnits[0] == "" {
		return r.leadingPoint(req, partsForUnits[1:])
	}
	if utf8.RuneCountInString(partsForUnits[0]) > 1 && leadingZeroDigitRun(partsForUnits[0]) {
		if req.rangeZeroDecimal && len(partsForUnits) == 2 {
			return r.rangeLeadingZeroDecimal(req, partsForUnits[0], partsForUnits[1])
		}
		return r.connectorDecimal(req)
	}
	if len(partsForUnits) == 2 && partsForUnits[1] != "" && strings.ContainsRune("〇零一二三四五六七八九０１２３４５６７８９", []rune(partsForUnits[1])[0]) && len(r.digitTypeRuns(strings.ReplaceAll(req.Number, "．", ""))) == 1 && strings.ContainsRune(partsForUnits[0], '零') && !strings.ContainsAny(partsForUnits[0], "十百千万億兆京垓") {
		return r.literalZeroDecimal(req, partsForUnits[0], partsForUnits[1])
	}
	unitPrefix := strings.ContainsAny(partsForUnits[0], "十百千万億兆京垓")
	if unitPrefix && leadingZeroDigitRunPrefix(partsForUnits[0]) {
		return r.connectorDecimal(req)
	}
	for _, fraction := range partsForUnits[1:] {
		if strings.ContainsAny(fraction, "十百千万億兆京垓") {
			rs := []rune(fraction)
			if len(partsForUnits) != 2 || len(rs) < 2 || !strings.ContainsRune("十百千"+string(bigUnits), rs[len(rs)-1]) {
				return r.connectorDecimal(req)
			}
			if _, small := smallUnits[rs[len(rs)-1]]; small && (unitPrefix && strings.ContainsAny(req.Number, "０１２３４５６７８９") || strings.ContainsRune(partsForUnits[0], '零')) {
				return r.connectorDecimal(req)
			}
			for _, ch := range rs[:len(rs)-1] {
				if _, digit := kanjiDigits[ch]; !digit {
					return r.connectorDecimal(req)
				}
			}
			// The Eng unit-fraction pattern uses lower-case n for its
			// prefix: written units there select the generic connector.
			if unitPrefix && strings.ContainsAny(req.Number, "０１２３４５６７８９") {
				return r.connectorDecimal(req)
			}
		}
	}
	if strings.Count(req.Number, "．") > 2 || len(r.digitTypeRuns(strings.ReplaceAll(req.Number, "．", ""))) > 1 {
		return r.connectorDecimal(req)
	}
	if strings.Count(req.Number, "．") == 2 {
		for _, c := range req.Number {
			if c != '．' && (c < '０' || c > '９') {
				return r.connectorDecimal(req)
			}
		}
		return r.doubleDecimal(req)
	}
	parts := strings.SplitN(req.Number, "．", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("unsupported decimal %q", req.Number)
	}
	if unitPrefix {
		return r.unitPrefixDecimal(req, parts[0], parts[1])
	}
	v, err := Value(parts[0])
	if err != nil {
		return nil, err
	}
	chunks := r.numberChunks(v)
	var groups []Group
	for _, c := range chunks[:len(chunks)-1] {
		g := r.accentOf(c)
		groups = append(groups, g)
	}
	last := chunks[len(chunks)-1]
	read, sound := r.tenWord(last)
	dm := MoraCount(sound)
	if last.TkKey == last.Key && last.Unit == "" && len(last.Parts) == 0 {
		// a single ones digit forms its own group
	}
	groups = append(groups, group(read+"テン", sound+"テン", dm-1, dm+2))
	if len(groups) > 0 && len(chunks) > 1 {
		r.flatBeforeMore(chunks[len(chunks)-2], &groups[len(groups)-2])
	}
	fracGroups, err := r.decimalFraction(req, parts[1])
	if err != nil {
		return nil, err
	}
	return []Reading{{Groups: append(groups, fracGroups...)}}, nil
}

func leadingZeroDigitRunPrefix(number string) bool {
	return strings.HasPrefix(number, "０") || strings.HasPrefix(number, "〇")
}

// Read the actual unit fragment before the point. Value would invent イチ
// for a bare 万 and collapse the paired 一零 in 兆一零 into a value.
func (r *Rules) unitPrefixDecimal(req Request, integer, fraction string) ([]Reading, error) {
	rd, err := r.connectorFragment(Request{Number: integer, NoSpecial: true})
	if err != nil {
		return nil, err
	}
	groups := rd[0].Groups
	for i := 0; i < len(groups)-1; i++ {
		c := chunk{Read: groups[i].Read, Sound: groups[i].Sound}
		for _, unit := range r.Units {
			c.literalUnitBoundary = c.literalUnitBoundary || groups[i+1].Sound == unit.Sound
		}
		r.flatBeforeMore(c, &groups[i])
	}
	last := groups[len(groups)-1]
	read, sound, fall := last.Read, last.Sound, last.Down
	if strings.ContainsRune(integer, '零') {
		for _, w := range r.Accents {
			if w.Sound != "" && strings.HasPrefix(sound, w.Sound) && len(w.Flag) > 0 {
				fall = int(w.Flag[0] - '0')
				break
			}
		}
	} else {
		cs, e := r.numericSoundChunks(sound)
		if e != nil || len(cs) != 1 {
			return nil, fmt.Errorf("unsupported decimal unit fragment %q", integer)
		}
		read, sound = r.tenWord(cs[0])
		fall = MoraCount(sound) - 1
	}
	if strings.ContainsRune(integer, '零') && strings.HasSuffix(integer, "十") && strings.HasSuffix(sound, "ジュー") {
		read, sound = geminate(read), geminate(sound)
	}
	attachedUnit := false
	for _, unit := range bigUnits {
		u := r.Units[string(unit)]
		attachedUnit = attachedUnit || utf8.RuneCountInString(integer) > 1 && u.Sound != "" && strings.HasSuffix(last.Sound, u.Sound)
	}
	if attachedUnit && !r.separateDecimalConnector(last) {
		attachedUnit = false
		fall = last.Down
	}
	if attachedUnit {
		groups = append(groups, group("テン", "テン", 0, 2), Group{Read: "ｂ", Pause: true})
	} else {
		groups[len(groups)-1] = group(read+"テン", sound+"テン", fall, MoraCount(sound)+2)
	}
	frac, err := r.decimalFraction(req, fraction)
	if err != nil {
		return nil, err
	}
	return []Reading{{Groups: append(groups, frac...)}}, nil
}

func (r *Rules) decimalFraction(req Request, fraction string) ([]Group, error) {
	chars := []rune(fraction)
	if len(chars) >= 2 && strings.ContainsRune("十百千"+string(bigUnits), chars[len(chars)-1]) {
		return r.unitFraction(req, chars[:len(chars)-1], string(chars[len(chars)-1]))
	}
	// Fractional digits bypass full integer-word accents. Ranges retain
	// the numeric key rebuilt for their final counter.
	req.unkeyedDigits = !req.rangeTail
	var frac []rune
	for _, c := range fraction {
		if _, ok := kanjiDigits[c]; !ok {
			return nil, fmt.Errorf("unsupported decimal digit %q", c)
		}
		frac = append(frac, c)
	}
	var groups []Group
	// the fraction
	var counterOnLast bool
	n := len(frac)
	pairs := n / 2
	if n%2 == 1 && req.Counter != "" {
		counterOnLast = true
	}
	for i := 0; i < pairs; i++ {
		a, b := fractionPair[digitRune(frac[2*i])], fractionPair[digitRune(frac[2*i+1])]
		if 2*i+2 == n && req.Counter != "" {
			// Ordinary fractional pairs have no final-digit key. A range
			// rebuilds the counter with that key, retaining fractional
			// sounds unless a changed digit must become a separate word.
			c := chunk{Read: a.Read + b.Read, Sound: a.Sound + b.Sound}
			if req.rangeTail {
				c.Key, c.TkKey = digitKanji[kanjiDigits[frac[2*i+1]]], digitKanji[kanjiDigits[frac[2*i+1]]]
				single := chunk{Read: b.Read, Sound: b.Sound, Key: c.Key, TkKey: c.TkKey}
				changed, err := r.withCounter(req, []chunk{single}, false)
				if err != nil {
					return nil, err
				}
				if !strings.HasPrefix(changed[0].Groups[0].Sound, b.Sound) {
					groups = append(groups, r.accentOf(chunk{Read: a.Read, Sound: a.Sound}))
					groups = append(groups, changed[0].Groups...)
					continue
				}
			}
			rs, err := r.withCounter(req, []chunk{c}, !req.rangeTail)
			if err != nil {
				return nil, err
			}
			groups = append(groups, rs[0].Groups...)
		} else {
			groups = append(groups, group(a.Read+b.Read, a.Sound+b.Sound, 3, 4))
		}
		if 2*i+2 < n {
			groups = append(groups, Group{Read: "ｂ", Pause: true})
		}
	}
	if n%2 == 1 {
		key := digitKanji[kanjiDigits[frac[n-1]]]
		w := r.digitWordByKey(key)
		if key == "〇" {
			w = Digit{Word: Word{Read: "レイ", Sound: "レー"}}
		}
		if counterOnLast {
			c := chunk{Read: w.Read, Sound: w.Sound}
			// Ordinary 1043f0 fractional tails reach 10f0a0 without
			// integer digit adjustments, including a lone final digit.
			// Ranges rebuild the tail with its numeric counter key.
			if req.rangeTail {
				c.Key, c.TkKey = key, key
			}
			rs, err := r.withCounter(Request{Number: key, Counter: req.Counter, CounterFall: req.CounterFall, CounterMoras: req.CounterMoras, CounterHasInfo: req.CounterHasInfo, unkeyedDigits: req.unkeyedDigits}, []chunk{c}, !req.rangeTail)
			if err != nil {
				return nil, err
			}
			groups = append(groups, rs[0].Groups...)
		} else {
			groups = append(groups, r.accentOf(chunk{Read: w.Read, Sound: w.Sound, Key: key, TkKey: key}))
		}
	}
	return groups, nil
}

func digitRune(c rune) rune { return []rune(digitKanji[kanjiDigits[c]])[0] }

// tenWord is the reading of a chunk that is followed by テン.
func (r *Rules) tenWord(c chunk) (read, sound string) {
	if len(c.Parts) == 2 { // ジュウ + digit (11..19)
		tens := c.Parts[0]
		w := beforeTen[c.Parts[1].Key]
		sound = tens.Sound + w.Sound
		read = tens.Sound + w.Sound
		if t, ok := r.BySound[sound]; ok {
			read = t.Read
		}
		return read, sound
	}
	if c.Unit == "" && c.Key == c.TkKey {
		if w, ok := beforeTen[c.Key]; ok {
			return w.Read, w.Sound
		}
	}
	read, sound = r.chunkWord(c)
	// 十 takes ッ (ジュッテン); 百 and 千 do not
	if c.Key == "十" && c.Unit == "" {
		read, sound = geminate(read), geminate(sound)
	}
	return
}

// The general connector path reads script runs as numeric values. It serves
// mixed scripts and point series outside the single/double decimal patterns;
// テン does not lengthen or geminate the preceding digit.
func (r *Rules) ReadConnectorPoint(parts []string, last Request) ([]Reading, error) {
	last.Number = strings.Join(parts, "．")
	last.NoSpecial = true
	return r.connectorDecimal(last)
}

// Label prefixes such as 図 accept a three-part decimal in either script.
// The unprefixed Jap pattern instead uses the generic connector path.
func (r *Rules) ReadLabelDecimal(parts []string, last Request) ([]Reading, error) {
	if len(parts) == 2 {
		if _, unit := r.Units[parts[1]]; unit {
			value, err := Value(parts[0])
			if err != nil {
				return nil, err
			}
			cs := r.numberChunks(value)
			var gs []Group
			for i, c := range cs {
				if i == len(cs)-1 {
					read, sound := r.tenWord(c)
					gs = append(gs, group(read+"テン", sound+"テン", MoraCount(sound)-1, MoraCount(sound)+2))
				} else {
					g := r.accentOf(c)
					r.flatBeforeMore(c, &g)
					gs = append(gs, g)
				}
			}
			last.Number = parts[1]
			tail, err := r.Read(last)
			if err != nil {
				return nil, err
			}
			return []Reading{{Groups: append(gs, tail[0].Groups...)}}, nil
		}
	}
	last.Number = strings.Join(parts, "．")
	if len(parts) == 3 {
		return r.doubleDecimal(last)
	}
	return r.decimal(last)
}

func (r *Rules) connectorDecimal(req Request) ([]Reading, error) {
	parts := strings.Split(req.Number, "．")
	var groups []Group
	for pi, part := range parts {
		if part == "" {
			return nil, fmt.Errorf("unsupported decimal %q", req.Number)
		}
		separatePoint := false
		runs := r.digitTypeRuns(part)
		if strings.ContainsAny(part, "十百千"+string(bigUnits)) {
			// A written unit resets 100a90's digit-script boundary. Let
			// the generic unit reader concatenate its fragments before
			// accent lookup (十８十八 -> ジューハチ / ジューハチ).
			// The unit reader retains literal-zero pairs while joining
			// following unit and digit fragments for table lookup.
			runs = []string{part}
		}
		for ri, run := range runs {
			// 100a90 dispatches a Japanese run containing 零 to the
			// paired digit reader before crossing into Arabic digits.
			// The general point path retains that reading and its s0
			// boundary; evaluating 一零 would incorrectly produce 十.
			if strings.ContainsRune(run, '零') && ri < len(runs)-1 {
				rs, err := r.read(Request{Number: run}, true)
				if err != nil {
					return nil, err
				}
				groups = append(groups, rs[0].Groups...)
				groups = append(groups, Group{Read: "ｂ", Pause: true})
				continue
			}
			lastRun := pi == len(parts)-1 && ri == len(runs)-1
			fragmentReq := Request{Number: run, NoSpecial: true}
			if lastRun {
				fragmentReq = req
				fragmentReq.Number, fragmentReq.NoSpecial = run, true
			}
			var readings []Reading
			var err error
			if pi > 0 && writtenLargeUnits(run) {
				readings, err = r.connectorUnitWords(fragmentReq)
			} else {
				readings, err = r.connectorFragment(fragmentReq)
			}
			if err != nil {
				return nil, err
			}
			for gi, g := range readings[0].Groups {
				// A digit-script boundary keeps the final table accent in a
				// unit-free fraction. Written units retain continuation rules.
				if !g.Pause && gi == len(readings[0].Groups)-1 && !lastRun && (pi == 0 || ri == len(runs)-1 || strings.ContainsAny(part, "十百千"+string(bigUnits))) {
					r.flatBeforeMore(chunk{Read: g.Read, Sound: g.Sound}, &g)
				}
				if gi == len(readings[0].Groups)-1 && ri == len(runs)-1 && pi < len(parts)-1 {
					attachedUnit := false
					for _, unit := range bigUnits {
						u := r.Units[string(unit)]
						// The first bare unit takes テン in the same word, even
						// inside a connector stream. A unit after an earlier point
						// or written pause keeps テン as a separate word.
						attachedUnit = attachedUnit || (pi > 0 || req.writtenPause || utf8.RuneCountInString(run) > 1) && u.Sound != "" && strings.HasSuffix(g.Sound, u.Sound) && (pi > 0 || req.writtenPause || r.separateDecimalConnector(g))
					}
					if attachedUnit {
						groups = append(groups, g)
						g = group("テン", "テン", 0, 2)
						separatePoint = true
					} else {
						fall := g.Down
						for _, w := range r.Accents {
							if w.Sound != "" && strings.HasPrefix(g.Sound, w.Sound) && len(w.Flag) > 0 {
								fall = int(w.Flag[0] - '0')
								break
							}
						}
						g = group(g.Read+"テン", g.Sound+"テン", fall, g.Moras+2)
					}
				}
				groups = append(groups, g)
			}
			if ri < len(runs)-1 {
				groups = append(groups, Group{Read: "ｂ", Pause: true})
			}
		}
		if pi < len(parts)-1 {
			if separatePoint {
				groups = append(groups, Group{Read: "ｂ", Pause: true})
			} else {
				groups = append(groups, Group{Pause: true, Silent: true})
			}
		}
	}
	return []Reading{{Groups: groups}}, nil
}

// The connector's post-point word rebuild separates the first bare large unit
// (1．万万). The remaining unit stream still uses integer table grouping,
// including its unit attachment (1．万万万 is マン / マンマン).
func writtenLargeUnits(number string) bool {
	if utf8.RuneCountInString(number) < 2 {
		return false
	}
	for _, c := range number {
		if !strings.ContainsRune(string(bigUnits), c) {
			return false
		}
	}
	return true
}

func (r *Rules) connectorUnitWords(req Request) ([]Reading, error) {
	chars := []rune(req.Number)
	if len(r.digitTypeRuns(req.Number)) > 1 {
		head := r.Units[string(chars[0])]
		rest := req
		rest.Number = string(chars[1:])
		tail, err := r.connectorFragment(rest)
		if err != nil {
			return nil, err
		}
		g := r.accentOf(chunk{Read: head.Read, Sound: head.Sound, Key: string(chars[0]), TkKey: string(chars[0]), numericFragment: true})
		tail[0].Groups = append([]Group{g}, tail[0].Groups...)
		return tail, nil
	}

	end := 0
	for end < len(chars) && strings.ContainsRune(string(bigUnits), chars[end]) {
		end++
	}
	if end > 1 && end < len(chars) && strings.ContainsRune("０〇零", chars[end]) {
		// The next 10b010 fragment starts at the zero digit. Evaluating
		// it together with the preceding unit would skip that zero.
		head, err := r.connectorUnitWords(Request{Number: string(chars[:end]), NoSpecial: true})
		if err != nil {
			return nil, err
		}
		req.Number = string(chars[end:])
		tail, err := r.connectorFragment(req)
		if err != nil {
			return nil, err
		}
		return []Reading{{Groups: append(head[0].Groups, tail[0].Groups...)}}, nil
	}
	key := string(chars[0])
	u := r.Units[key]
	chunks := []chunk{{Read: u.Read, Sound: u.Sound, Key: key, TkKey: key, numericFragment: true}}
	sound, err := r.integerFragmentSound(string(chars[1:]))
	if err != nil {
		return nil, err
	}
	tail, err := r.numericSoundChunks(sound)
	if err != nil {
		return nil, err
	}
	chunks = append(chunks, tail...)
	if req.Counter != "" {
		return r.withCounter(req, chunks, false)
	}
	var groups []Group
	for _, c := range chunks {
		groups = append(groups, r.accentOf(c))
	}
	return []Reading{{Groups: groups}}, nil
}

// A fragment after a written pause starts a new table word at its first
// large unit, even when digits follow. This does not apply to the first
// fragment of the expression or to the ordinary ノ/カラ patterns.
func (r *Rules) writtenPauseFragment(req Request) ([]Reading, error) {
	req.writtenPause = true
	chars := []rune(req.Number)
	if len(chars) > 1 && strings.ContainsRune(string(bigUnits), chars[0]) && strings.ContainsRune(string(bigUnits), chars[1]) {
		return r.connectorUnitWords(req)
	}
	return r.connectorFragment(req)
}

// A literal unit at the start is dispatched before the following paired 零
// digits. Keep their group intact instead of evaluating the numeric value.
func (r *Rules) connectorFragment(req Request) ([]Reading, error) {
	req.connectorStream = true
	// A literal unit inside 100a90 retains its numeric-fragment counter
	// mode. Standalone 千 instead has the separate bare-unit adjustment.
	if u, ok := r.Units[req.Number]; ok && u.Type == "Unit" && utf8.RuneCountInString(req.Number) == 1 {
		c := chunk{Read: u.Read, Sound: u.Sound, Key: req.Number, TkKey: req.Number, numericFragment: true}
		if req.Counter != "" {
			return r.withCounter(req, []chunk{c}, false)
		}
		return []Reading{{Groups: []Group{r.accentOf(c)}}}, nil
	}
	// 100a90 passes the connector's digit run to 10b010, rather than the
	// whole-number zero-skipping/paired-digit patterns. Keep its short final
	// digit and numeric counter key (1．〇五 is ゼロゴ, not ゼロゴー).
	if leadingZeroDigitRun(req.Number) && utf8.RuneCountInString(req.Number) > 1 {
		return r.read(req, true)
	}
	return r.unitZeroTail(req)
}

func (r *Rules) unitZeroTail(req Request) ([]Reading, error) {
	chars := []rune(req.Number)
	i := 0
	for i < len(chars) {
		if u, ok := r.Units[string(chars[i])]; !ok || u.Type != "Unit" {
			break
		}
		i++
	}
	if i == 0 {
		if z := strings.IndexRune(req.Number, '零'); z > 0 {
			prefix := []rune(req.Number[:z])
			if strings.ContainsRune("十百千"+string(bigUnits), prefix[len(prefix)-1]) {
				i = len(prefix)
			}
		}
	}
	if i == 0 || i == len(chars) || !strings.ContainsRune(string(chars[i:]), '零') {
		return r.Read(req)
	}
	prefix, err := r.Read(Request{Number: string(chars[:i]), NoSpecial: true, reiTail: true})
	if err != nil {
		return nil, err
	}
	req.Number = string(chars[i:])
	tail, err := r.Read(req)
	if err != nil {
		return nil, err
	}
	gs := prefix[0].Groups
	for j := range gs {
		r.flatBeforeMore(chunk{Read: gs[j].Read, Sound: gs[j].Sound}, &gs[j])
	}
	if len(gs) > 0 && len(tail[0].Groups) > 0 {
		if joined, ok := r.joinNumericPairPrefix(gs[len(gs)-1], tail[0].Groups[0]); ok {
			out := append(append([]Group(nil), gs[:len(gs)-1]...), joined...)
			out = append(out, tail[0].Groups[1:]...)
			return []Reading{{Groups: out}}, nil
		}
	}
	return []Reading{{Groups: append(gs, tail[0].Groups...)}}, nil
}

// joinNumericPairPrefix preserves a table word crossing into a paired
// literal-zero group, including its long vowel and remaining pair boundary.
func (r *Rules) joinNumericPairPrefix(head, tail Group) ([]Group, bool) {
	if head.Pause || tail.Pause {
		return nil, false
	}
	sound := head.Sound + tail.Sound
	for _, w := range r.Accents {
		if w.Sound == "" || len(w.Sound) <= len(head.Sound) || !strings.HasPrefix(sound, w.Sound) {
			continue
		}
		g := group(w.Read, w.Sound, w.Fall, w.Moras)
		rest := strings.TrimPrefix(sound, w.Sound)
		extended := false
		for strings.HasPrefix(rest, "ー") {
			g.Read += "ー"
			g.Sound += "ー"
			g.Moras++
			rest = strings.TrimPrefix(rest, "ー")
			extended = true
		}
		if extended && len(w.Flag) > 0 {
			g.Down = int(w.Flag[0] - '0')
			g.Up = rise(g.Down, g.Moras)
		}
		var suffix []Group
		cs, err := r.numericSoundChunks(rest)
		if err != nil {
			if word, ok := r.BySound[rest]; ok {
				suffix = []Group{group(word.Read, word.Sound, word.Fall, word.Moras)}
			} else {
				return nil, false
			}
		} else {
			for i, c := range cs {
				part := r.accentOf(c)
				if i+1 < len(cs) {
					r.flatBeforeMore(c, &part)
				}
				suffix = append(suffix, part)
			}
		}
		return append([]Group{g}, suffix...), true
	}
	return nil, false
}

// Exactly two points retain the first decimal's fractional digit groups.
// The second point appends テン without another digit sound change.
func (r *Rules) doubleDecimal(req Request) ([]Reading, error) {
	parts := strings.Split(req.Number, "．")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return nil, fmt.Errorf("unsupported decimal %q", req.Number)
	}
	prefix, err := r.decimal(Request{Number: parts[0] + "．" + parts[1]})
	if err != nil {
		return nil, err
	}
	groups := prefix[0].Groups
	last := &groups[len(groups)-1]
	fall := last.Down
	// Native accent lookup accepts the longest leading table word.
	for _, w := range r.Accents {
		if strings.HasPrefix(last.Sound, w.Sound) && len(w.Flag) > 0 {
			fall = int(w.Flag[0] - '0')
			break
		}
	}
	*last = group(last.Read+"テン", last.Sound+"テン", fall, last.Moras+2)
	tail, err := r.decimalFraction(req, parts[2])
	if err != nil {
		return nil, err
	}
	return []Reading{{Groups: append(groups, tail...)}}, nil
}

// 1043f0 reads paired digits, but the last digit before a written unit is
// short. Its final sound segment uses the accent table's digit-pair and unit
// flags, including longer entries such as イチニセン and ニチョー.
func (r *Rules) unitFraction(req Request, digits []rune, unit string) ([]Group, error) {
	for _, c := range digits {
		if _, ok := kanjiDigits[c]; !ok {
			return nil, fmt.Errorf("unsupported decimal digit %q", c)
		}
	}
	var groups []Group
	for len(digits) > 2 {
		a, b := fractionPair[digitRune(digits[0])], fractionPair[digitRune(digits[1])]
		groups = append(groups, group(a.Read+b.Read, a.Sound+b.Sound, 3, 4), Group{Read: "ｂ", Pause: true})
		digits = digits[2:]
	}
	var sound string
	if len(digits) == 2 {
		sound = fractionPair[digitRune(digits[0])].Sound
	}
	last := r.digitWordByKey(digitKanji[kanjiDigits[digits[len(digits)-1]]])
	digit := digits[len(digits)-1]
	if kanjiDigits[digit] == 0 {
		last.Read, last.Sound = fractionPair['〇'].Read, fractionPair['〇'].Sound
	}
	if scale, small := smallUnits[[]rune(unit)[0]]; small && r.Digits[string(digit)].Type == "Jap" && kanjiDigits[digit] > 1 {
		for _, c := range r.groupChunks(kanjiDigits[digit] * int(scale)) {
			sound += c.Sound
		}
	} else {
		sound += last.Sound + r.Units[unit].Sound
	}
	tail := r.fractionAccentGroups(sound)
	if req.Counter != "" {
		final := tail[len(tail)-1]
		key := unit
		if _, small := smallUnits[[]rune(unit)[0]]; small {
			key = digitKanji[kanjiDigits[digits[len(digits)-1]]]
		}
		rd, err := r.withCounter(req, []chunk{{Read: final.Read, Sound: final.Sound, Key: unit, TkKey: key}}, false)
		if err != nil {
			return nil, err
		}
		tail = append(tail[:len(tail)-1], rd[0].Groups...)
	}
	return append(groups, tail...), nil
}

func (r *Rules) fractionAccentGroups(sound string) []Group {
	type word struct {
		group Group
		flags string
	}
	var words []word
	for sound != "" {
		var match *AccentWord
		for i := range r.Accents {
			if r.Accents[i].Sound != "" && strings.HasPrefix(sound, r.Accents[i].Sound) {
				match = &r.Accents[i]
				break
			}
		}
		if match == nil {
			g := group(sound, sound, 0, MoraCount(sound))
			g.Up = 0
			words = append(words, word{g, "00000"})
			break
		}
		parts := splitMoras(sound, []int{match.Moras, MoraCount(sound) - match.Moras})
		words = append(words, word{group(parts[0], parts[0], match.Fall, MoraCount(parts[0])), match.Flag})
		sound = parts[1]
	}
	var groups []Group
	for i := 0; i < len(words); i++ {
		w := words[i]
		g := w.group
		if i+1 < len(words) && len(w.flags) >= 5 && w.flags[4] == '1' && len(words[i+1].flags) >= 5 && words[i+1].flags[4] == '1' {
			next := words[i+1]
			g = group(g.Read+next.group.Read, g.Sound+next.group.Sound, g.Moras+int(next.flags[0]-'0'), g.Moras+next.group.Moras)
			i++
		}
		if i+1 < len(words) && len(words[i+1].flags) >= 4 && words[i+1].flags[3] == '1' {
			next := words[i+1].group
			fall := g.Moras
			if next.Down > 0 {
				fall += next.Down
			}
			g = group(g.Read+next.Read, g.Sound+next.Sound, fall, g.Moras+next.Moras)
			i++
		}
		groups = append(groups, g)
	}
	return groups
}

// leadingPoint is the generic 100a90/1043f0 stream selected by multiple
// points with no integer prefix. It uses short lone digits, paired digits,
// and no beforeTen lengthening/gemination.
func (r *Rules) leadingPoint(req Request, parts []string) ([]Reading, error) {
	groups := []Group{group("テン", "テン", 0, 2), {Pause: true, Silent: true}}
	for i, part := range parts {
		tail := Request{Number: part}
		if i == len(parts)-1 {
			tail = req
			tail.Number = part
		}
		gs, err := r.decimalFraction(tail, part)
		if err != nil {
			if _, unit := r.Units[part]; !unit || i != 0 || tail.Counter != "" {
				return nil, err
			}
			// 1043f0 attaches an initial bare unit to the preceding テン.
			groups[0] = r.accentOf(chunk{Read: "テン", Sound: "テン", Unit: part})
			if i < len(parts)-1 {
				groups = append(groups, group("テン", "テン", 0, 2), Group{Pause: true, Silent: true})
			}
			continue
		}
		if len(gs) == 0 {
			return nil, fmt.Errorf("empty point fragment")
		}
		if i < len(parts)-1 {
			g := &gs[len(gs)-1]
			fall := g.Down
			for _, w := range r.Accents {
				if w.Sound != "" && strings.HasPrefix(g.Sound, w.Sound) && len(w.Flag) > 0 {
					fall = int(w.Flag[0] - '0')
					break
				}
			}
			*g = group(g.Read+"テン", g.Sound+"テン", fall, g.Moras+2)
		}
		groups = append(groups, gs...)
		if i < len(parts)-1 {
			groups = append(groups, Group{Pause: true, Silent: true})
		}
	}
	return []Reading{{Groups: groups}}, nil
}

func (r *Rules) rangeLeadingZeroDecimal(req Request, integer, fraction string) ([]Reading, error) {
	chars := []rune(integer)
	n := 0
	for n < len(chars) && (chars[n] == '〇' || chars[n] == '０') {
		n++
	}
	zero, err := r.ReadDigits([]string{string(chars[:n])}, false)
	if err != nil {
		return nil, err
	}
	prefix := zero[0].Groups
	for i := range prefix {
		prefix[i].Read = strings.ReplaceAll(prefix[i].Read, "ゼロ", "レイ")
		prefix[i].Sound = strings.ReplaceAll(prefix[i].Sound, "ゼロ", "レー")
	}
	if n == len(chars) {
		g := &prefix[len(prefix)-1]
		g.Read += "テン"
		g.Sound += "テン"
		g.Moras += 2
		g.Down = 1
		g.Up = 1
		tail, err := r.decimalFraction(req, fraction)
		if err != nil {
			return nil, err
		}
		return []Reading{{Groups: append(prefix, tail...)}}, nil
	}
	tailReq := req
	tailReq.Number = string(chars[n:]) + "．" + fraction
	tail, err := r.decimal(tailReq)
	if err != nil {
		return nil, err
	}
	v, err := Value(string(chars[n:]))
	if err != nil {
		return nil, err
	}
	if n%2 == 1 && v.IsInt64() && v.Int64() < 10 && v.Int64() != 1 && v.Int64() != 8 {
		z := prefix[len(prefix)-1]
		g := &tail[0].Groups[0]
		g.Read = z.Read + g.Read
		g.Sound = z.Sound + g.Sound
		g.Moras += z.Moras
		g.Down = 1
		g.Up = 1
		prefix = prefix[:len(prefix)-1]
	}
	tail[0].Groups = append(prefix, tail[0].Groups...)
	return tail, nil
}

func (r *Rules) literalZeroDecimal(req Request, integer, fraction string) ([]Reading, error) {
	prefix, err := r.ReadDigits([]string{integer}, false)
	if err != nil {
		return nil, err
	}
	gs := prefix[0].Groups
	chars := []rune(integer)
	digit := chars[len(chars)-1]
	g := &gs[len(gs)-1]
	if (digit == '五' || digit == '二') && len(chars)%2 == 0 {
		gs = append(gs, Group{Read: "ー", Sound: "ー", Up: 2, Moras: 1}, Group{Read: "テン", Sound: "テン", Up: 2, Moras: 2})
	} else if digit == '一' || digit == '八' {
		base := r.digitWordByKey(string(digit))
		before := beforeTen[string(digit)]
		if len(chars)%2 == 0 {
			g.Read = strings.TrimSuffix(g.Read, base.Read)
			g.Sound = strings.TrimSuffix(g.Sound, base.Sound)
			g.Moras = MoraCount(g.Sound)
			g.Down = 1
			g.Up = 1
			gs = append(gs, group(before.Read+"テン", before.Sound+"テン", 1, MoraCount(before.Sound)+2))
		} else {
			*g = group(before.Read+"テン", before.Sound+"テン", 1, MoraCount(before.Sound)+2)
		}
	} else {
		if digit == '五' || digit == '二' {
			g.Read += "ー"
			g.Sound += "ー"
			g.Moras++
		}
		g.Read += "テン"
		g.Sound += "テン"
		g.Moras += 2
		g.Down = 1
		g.Up = 1
	}
	tail, err := r.decimalFraction(req, fraction)
	if err != nil {
		return nil, err
	}
	return []Reading{{Groups: append(gs, tail...)}}, nil
}
