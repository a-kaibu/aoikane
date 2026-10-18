package numeral

import (
	"fmt"
	"math/big"
	"strings"
	"unicode/utf8"
)

// Group is one accent group of a numeral word: its reading and pronunciation
// and the accent as the engine stores it (rise, fall, moras; a fall of 0 means
// a flat accent).
type Group struct {
	Read, Sound     string
	Up, Down, Moras int
	// Pause marks a short pause between groups (the engine writes it as ｂ in
	// the reading and "s0" in the accent); it has no pronunciation of its own.
	Pause bool
	// Silent is a pause that is not written in the reading (times).
	Silent bool
	// Strong marks the longer pause between blocks of a digit string ("s1").
	Strong bool
	fixed  bool
}

// Reading is one candidate reading of a numeral run.
type Reading struct {
	Name       string // candidate reading label, e.g. "ヒトリ２" for the second alternative
	Connection string // special-reading accent connection; empty means "*"
	Groups     []Group
}

// Request describes a numeral run: its characters (digits and units as they
// appear in the text) and an optional counter with the counter word's own
// accent taken from its dictionary entry.
type Request struct {
	compoundDigits bool
	// Paired 1043f0 digits with counter mode disabled skip full-word accents.
	unkeyedDigits bool
	// Literal ratio connectors with a counter use the generic point reader.
	genericDecimal bool
	// Range-tail decimals retain the final digit's counter sound change.
	rangeTail bool
	// The numeral is inside a generic connector stream, not a plain decimal.
	connectorStream bool
	// This fragment starts after a written numeric pause in the same stream.
	writtenPause bool
	// Preserve numeric fragment boundaries before a following literal 零.
	reiTail bool
	// Paired digits before a written large unit retain 1043f0's counter
	// adjustment even when unit assimilation delegates the nonzero tail.
	pairedUnit     bool
	Number         string
	Counter        string
	CounterFall    int
	CounterMoras   int
	CounterHasInfo bool
	// NoSpecial skips the [特殊読み] table (times, fractions).
	NoSpecial bool
}

const smallKana = "ァィゥェォャュョヮ"

// MoraCount counts the moras of a kana string.
func MoraCount(s string) int {
	n := 0
	for _, r := range s {
		if !strings.ContainsRune(smallKana, r) {
			n++
		}
	}
	return n
}

// splitMoras cuts s into the pieces of the given mora counts.
func splitMoras(s string, counts []int) []string {
	rs := []rune(s)
	var out []string
	pos := 0
	for _, c := range counts {
		start := pos
		n := 0
		for pos < len(rs) && n < c {
			pos++
			n++
			for pos < len(rs) && strings.ContainsRune(smallKana, rs[pos]) {
				pos++
			}
		}
		out = append(out, string(rs[start:pos]))
	}
	if pos < len(rs) && len(out) > 0 {
		out[len(out)-1] += string(rs[pos:])
	}
	return out
}

func rise(fall, moras int) int {
	if fall == 1 || moras == 1 {
		return 1
	}
	return 2
}

func group(read, sound string, fall, moras int) Group {
	return Group{Read: read, Sound: sound, Up: rise(fall, moras), Down: fall, Moras: moras}
}

var kanjiDigits = map[rune]int{'〇': 0, '零': 0, '一': 1, '二': 2, '三': 3, '四': 4, '五': 5, '六': 6, '七': 7, '八': 8, '九': 9,
	'０': 0, '１': 1, '２': 2, '３': 3, '４': 4, '５': 5, '６': 6, '７': 7, '８': 8, '９': 9}
var smallUnits = map[rune]int64{'十': 10, '百': 100, '千': 1000}
var bigUnits = []rune{'万', '億', '兆', '京', '垓'}

// Value evaluates a run of digits and units to a number.
func Value(s string) (*big.Int, error) {
	total := new(big.Int)
	section := int64(0) // value below 10000 collected so far
	cur := int64(-1)    // positional digits being collected, -1 when none
	flushUnit := func(mult *big.Int) {
		v := section
		if cur >= 0 {
			v += cur
		}
		cur = -1
		section = 0
		t := big.NewInt(v)
		if v == 0 {
			t.SetInt64(1)
		}
		total.Add(total, t.Mul(t, mult))
	}
	for _, r := range s {
		if d, ok := kanjiDigits[r]; ok {
			if cur < 0 {
				cur = 0
			}
			cur = cur*10 + int64(d)
			continue
		}
		if m, ok := smallUnits[r]; ok {
			c := int64(1)
			if cur >= 0 {
				c = cur
			}
			section += c * m
			cur = -1
			continue
		}
		matched := false
		for i, u := range bigUnits {
			if r == u {
				mult := new(big.Int).Exp(big.NewInt(10000), big.NewInt(int64(i+1)), nil)
				flushUnit(mult)
				matched = true
				break
			}
		}
		if !matched {
			return nil, fmt.Errorf("unexpected %q in number", r)
		}
	}
	v := section
	if cur >= 0 {
		v += cur
	}
	return total.Add(total, big.NewInt(v)), nil
}

// chunk is a piece of the number reading with the kanji key of its last digit
// or unit (for the Tj/Tk rows).
type chunk struct {
	Read, Sound string
	Key         string // "一".."九", "十", "百", "千", "万"..., "〇": the last digit or unit (row of Tj)
	// TkKey is the row of Tk: the chunk's digit when it has one before a unit
	// (二 of 二十), otherwise Key.
	TkKey string
	// Unit is the group unit (万, 億, ...) that follows the chunk's number.
	Unit string
	// Parts are the two chunks of a merged 11..19 (ジュウ + digit).
	Parts []chunk
	// A bare unit inside 100a90's numeric stream uses the table Tk row;
	// the standalone bare-unit path has its own accent treatment.
	numericFragment bool
	// A complete accent-table word, including its written large unit.
	tableStem bool
	// This word ends before a multi-digit integer fragment. Its table accent
	// survives that fragment and its counter; a following single digit still
	// uses the usual continuation flag (八十一, 五十九).
	fragmentBoundary bool
	// A following bare written unit is not a positional-value continuation.
	literalUnitBoundary bool
}

var digitKanji = []string{"〇", "一", "二", "三", "四", "五", "六", "七", "八", "九"}

// f7a10 changes the final mora of 一/八/十 before 兆 and 京.
func attachLargeUnit(c chunk, unit string) chunk {
	if unit == "兆" || unit == "京" {
		for _, ending := range []string{"イチ", "ハチ", "ジュー"} {
			if strings.HasSuffix(c.Sound, ending) {
				c.Read, c.Sound = geminate(c.Read), geminate(c.Sound)
				break
			}
		}
	}
	c.Key, c.TkKey, c.Unit = unit, unit, unit
	return c
}

func (r *Rules) digitWord(d int) Digit { return r.Digits[digitKanji[d]] }

// groupChunks reads one four-digit group (1..9999) as chunks.
func (r *Rules) groupChunks(g int) []chunk {
	var out []chunk
	th, h, t, o := g/1000, g/100%10, g/10%10, g%10
	add := func(d int, unit string, key string) {
		u := r.Units[unit]
		dw := r.digitWord(d)
		read, sound := dw.Read+u.Read, dw.Sound+u.Sound
		if d == 1 {
			read, sound = u.Read, u.Sound
		}
		// rendaku and sokuon of the unit (サンビャク, ロッピャク, ハッピャク, サンゼン, ハッセン)
		switch {
		case unit == "百" && d == 3:
			read, sound = dw.Read+"ビャク", dw.Sound+"ビャク"
		case unit == "百" && d == 6:
			read, sound = "ロッピャク", "ロッピャク"
		case unit == "百" && d == 8:
			read, sound = "ハッピャク", "ハッピャク"
		case unit == "千" && d == 3:
			read, sound = dw.Read+"ゼン", dw.Sound+"ゼン"
		case unit == "千" && d == 8:
			read, sound = "ハッセン", "ハッセン"
		}
		tk := digitKanji[d]
		if d == 1 {
			tk = key
		}
		out = append(out, chunk{Read: read, Sound: sound, Key: key, TkKey: tk})
	}
	if th > 0 {
		add(th, "千", "千")
	}
	if h > 0 {
		add(h, "百", "百")
	}
	if t > 0 {
		add(t, "十", "十")
	}
	if o > 0 {
		dw := r.digitWord(o)
		out = append(out, chunk{Read: dw.Read, Sound: dw.Sound, Key: digitKanji[o], TkKey: digitKanji[o]})
	}
	// 11..19 read as one word (ジューイチ ...)
	if t == 1 && o > 0 && len(out) >= 2 {
		n := len(out)
		j := chunk{Read: out[n-2].Read + out[n-1].Read, Sound: out[n-2].Sound + out[n-1].Sound, Key: out[n-1].Key, TkKey: out[n-1].TkKey}
		if _, ok := r.BySound[j.Sound]; ok {
			j.Parts = []chunk{out[n-2], out[n-1]}
			out = append(out[:n-2], j)
		}
	}
	return out
}

// numberChunks reads a value as chunks, merging each group's unit (万, 億, ...)
// into the group's last chunk.
func (r *Rules) numberChunks(v *big.Int) []chunk {
	if v.Sign() == 0 {
		z := r.digitWord(0)
		return []chunk{{Read: z.Read, Sound: z.Sound, Key: "〇", TkKey: "〇"}}
	}
	ten4 := big.NewInt(10000)
	var groups []int
	rest := new(big.Int).Set(v)
	for rest.Sign() > 0 {
		m := new(big.Int)
		rest.DivMod(rest, ten4, m)
		groups = append(groups, int(m.Int64()))
	}
	var out []chunk
	for i := len(groups) - 1; i >= 0; i-- {
		g := groups[i]
		if g == 0 {
			continue
		}
		cs := r.groupChunks(g)
		if i > 0 {
			unit := string(bigUnits[i-1])
			if g == 1 {
				one := r.digitWord(1)
				cs = []chunk{attachLargeUnit(chunk{Read: one.Read, Sound: one.Sound}, unit)}
			} else {
				cs[len(cs)-1] = attachLargeUnit(cs[len(cs)-1], unit)
			}
		}
		out = append(out, cs...)
	}
	return out
}

// accentOf returns the accent group of a chunk: the table entry of its
// pronunciation, with the group unit (万, 億, ...) attached by the accent of the
// unit (an accented unit keeps its own fall, a flat one falls after the number).
func (r *Rules) accentOf(c chunk) Group {
	read, sound := c.Read, c.Sound
	fall, moras := MoraCount(sound), MoraCount(sound)
	if w, ok := r.BySound[sound]; ok {
		read, fall, moras = w.Read, w.Fall, w.Moras
	}
	if c.Unit != "" {
		u := r.Units[c.Unit]
		if w, ok := r.BySound[sound+u.Sound]; ok {
			return group(w.Read, sound+u.Sound, w.Fall, w.Moras)
		}
		dm := moras
		uw := r.BySound[u.Sound]
		if uw.Fall > 0 {
			fall = dm + uw.Fall
		} else {
			fall = dm
		}
		read, sound = read+u.Read, sound+u.Sound
		moras = dm + MoraCount(u.Sound)
	}
	return group(read, sound, fall, moras)
}

// flatBeforeMore applies the second flag digit of a table word: such a word
// (ゴジュウ, ロクジュウ, ...) loses its fall when more of the number follows.
func (r *Rules) flatBeforeMore(c chunk, g *Group) {
	// 100a90 retains the final word's accent before a multi-digit fragment.
	// Internal positional value chunks still apply the continuation flag.
	if c.Unit != "" || c.fragmentBoundary || c.literalUnitBoundary {
		return
	}
	if w, ok := r.BySound[c.Sound]; ok && len(w.Flag) > 1 && w.Flag[1] == '1' {
		g.Down = 0
		g.Up = rise(0, g.Moras)
	}
}

// ignoredSpecials are table entries bypassed by the native generic reader:
// 9分 uses キュウフン (2-2-4); 零分 dispatches 零 to 1043f0 and uses
// レーフン (2-2-4), rather than the literal イ in its table alternatives.
// 零秒 likewise uses the generic レービョー (2-2-4), not the special
// first candidate (1-1-4). The difference survives Kansai suffix joining.
var ignoredSpecials = map[string]bool{"９分": true, "九分": true, "零分": true, "零秒": true}

// Read converts a numeral run into its candidate readings.
func (r *Rules) Read(req Request) ([]Reading, error) {
	return r.read(req, false)
}

// A fragment is part of a longer numeral, so whole-number special and
// zero-skipping patterns must not match it independently.
func (r *Rules) read(req Request, fragment bool) ([]Reading, error) {
	if utf8.RuneCountInString(req.Number) == 0 {
		return nil, fmt.Errorf("empty number")
	}
	if !fragment && r.ZeroSkippingCounters[req.Counter] && leadingZeroDigitRun(req.Number) {
		length := utf8.RuneCountInString(req.Number)
		zero := string([]rune(req.Number)[0])
		req.Number = strings.TrimLeft(req.Number, zero)
		if req.Number == "" {
			req.Number = zero
			// A three-or-more-digit all-zero integer uses the generic
			// counter path. It cannot acquire a short special reading
			// merely because the zero-skipping pattern reduced its value.
			if length > 2 {
				req.NoSpecial = true
			}
		}
	}
	if sp, ok := r.Specials[req.Number+req.Counter]; ok && sp.plain() && !fragment && !req.NoSpecial && !ignoredSpecials[req.Number+req.Counter] {
		return r.special(sp)
	}
	if strings.Contains(req.Number, "．") {
		return r.decimal(req)
	}
	// An initial written unit followed by a literal 零 digit stream keeps
	// 1043f0's paired groups. Re-parsing its レー sound through the integer
	// table loses that group (and can reject the entire numeral).
	if chars := []rune(req.Number); len(chars) > 1 {
		if u, unit := r.Units[string(chars[0])]; unit && u.Type == "Unit" && strings.ContainsRune(req.Number, '零') {
			return r.unitZeroTail(req)
		}
	}
	// An integer fragment ending in a written unit is dispatched
	// before a literal 零 stream, just like an initial bare unit.
	if z := strings.IndexRune(req.Number, '零'); z > 0 {
		prefix := []rune(req.Number[:z])
		if strings.ContainsRune("十百千"+string(bigUnits), prefix[len(prefix)-1]) {
			return r.unitZeroTail(req)
		}
	}
	// A bare unit keeps its literal table reading (万 is マン, not イチマン).
	// The implicit-one rule belongs to positional values such as 一万.
	if u, ok := r.Units[req.Number]; ok && utf8.RuneCountInString(req.Number) == 1 {
		c := chunk{Read: u.Read, Sound: u.Sound, Key: req.Number, TkKey: req.Number}
		if req.Counter != "" {
			return r.withCounter(req, []chunk{c}, false)
		}
		return []Reading{{Groups: []Group{r.accentOf(c)}}}, nil
	}
	// 100a90 dispatches an initial unit separately from the following digits.
	// Preserve and concatenate those readings before numeric table lookup;
	// Value would invent an implicit one for 万５百 or add 百十 and ９１.
	if req.connectorStream && LeadingZeroSmallUnitCoefficient(req.Number) {
		return r.readMixedZeroUnits(req)
	}
	expanded, expansion := expandedUnitInteger(req.Number)
	if req.reiTail {
		expansion = false
	}
	if pairedZeroLargeUnit(req.Number) && (req.NoSpecial || req.genericDecimal || req.connectorStream) {
		// The complete coefficient/unit pattern cannot match an internal
		// fragment, such as the head of ００兆万 or the integer in 秒.
		expansion = false
	}
	charsForUnits := []rune(req.Number)
	for i, ch := range charsForUnits {
		if !strings.ContainsRune(string(bigUnits), ch) {
			continue
		}
		if i > 0 && i+1 < len(charsForUnits) && strings.ContainsRune(string(bigUnits), charsForUnits[i+1]) && leadingZeroDigitRun(string(charsForUnits[:i])) {
			head, err := r.Read(Request{Number: string(charsForUnits[:i+1]), NoSpecial: true})
			if err != nil {
				return nil, err
			}
			tailReq := req
			tailReq.Number, tailReq.NoSpecial = string(charsForUnits[i+1:]), true
			tail, err := r.connectorFragment(tailReq)
			if err != nil {
				return nil, err
			}
			return []Reading{{Groups: append(head[0].Groups, tail[0].Groups...)}}, nil
		}
		break
	}
	if !fragment && !expansion && strings.ContainsAny(req.Number, "十百千万億兆京垓") && !pairedZeroLargeUnit(req.Number) {
		return r.readMixedZeroUnits(req)
	}
	if !fragment && expansion || fragment && strings.ContainsAny(req.Number, "十百千万億兆京垓") && !pairedZeroLargeUnit(req.Number) {
		if !fragment && expansion {
			req.Number = expanded
		}
		sound, err := r.integerFragmentSound(req.Number)
		if err != nil {
			return nil, err
		}
		var zeroGroups []Group
		if !fragment && expansion && strings.HasPrefix(req.Number, "００") {
			tail := strings.TrimLeft(req.Number, "０")
			if tail == "" {
				return r.readLeadingZero(req, true)
			}
			zeros := req.Number[:len(req.Number)-len(tail)]
			paired, err := r.ReadDigits([]string{zeros}, false)
			if err != nil {
				return nil, err
			}
			zeroGroups = paired[0].Groups
			sound = strings.TrimPrefix(sound, strings.Repeat(r.digitWord(0).Sound, utf8.RuneCountInString(zeros)))
		}
		chunks, err := r.numericSoundChunks(sound)
		if err != nil {
			return nil, err
		}
		if req.Counter != "" {
			rd, err := r.withCounter(req, chunks, false)
			if err == nil {
				rd[0].Groups = append(zeroGroups, rd[0].Groups...)
			}
			return rd, err
		}
		groups := zeroGroups
		for i, c := range chunks {
			g := r.accentOf(c)
			if i < len(chunks)-1 {
				r.flatBeforeMore(c, &g)
			}
			groups = append(groups, g)
		}
		return []Reading{{Groups: groups}}, nil
	}
	if strings.ContainsRune(req.Number, '零') && strings.ContainsAny(req.Number, "十百千"+string(bigUnits)) {
		return r.readMixedZeroUnits(req)
	}
	// 零 retains its table reading レイ/レー. In a Japanese digit sequence
	// it uses the paired digit reader rather than losing zeroes in Value.
	if strings.ContainsRune(req.Number, '零') && len(r.digitTypeRuns(req.Number)) == 1 {
		var parts []string
		start := 0
		for i, c := range req.Number {
			if _, digit := kanjiDigits[c]; !digit {
				if i > start {
					parts = append(parts, req.Number[start:i])
				}
				parts = append(parts, string(c))
				start = i + utf8.RuneLen(c)
			}
		}
		if len(parts) > 0 {
			if start < len(req.Number) {
				parts = append(parts, req.Number[start:])
			}
			return r.readZeroParts(req, parts)
		}
		if utf8.RuneCountInString(req.Number) > 1 {
			return r.readLeadingZero(req, fragment || req.rangeTail)
		}
	}
	// The original reads a digit run with a leading zero as a
	// sequence of digits. Evaluating its numeric value would lose that zero.
	if leadingZeroDigitRun(req.Number) && utf8.RuneCountInString(req.Number) >= 2 {
		return r.readLeadingZero(req, fragment)
	}
	// A zero followed by a large unit keeps that unit (０万 is ゼロマン).
	// It is not the implicit one used for a bare 万.
	chars := []rune(req.Number)
	// The paired digit groups survive a following large unit: ０３万 is
	// ゼロサンマン (one group), not ゼロ / サンマン.
	if len(chars) >= 3 && strings.ContainsRune(string(bigUnits), chars[len(chars)-1]) {
		prefix := string(chars[:len(chars)-1])
		if leadingZeroDigitRun(prefix) {
			req.pairedUnit = true
			readings, err := r.readLeadingZero(Request{Number: prefix}, true)
			if err != nil {
				return nil, err
			}
			groups := readings[0].Groups
			last := groups[len(groups)-1]
			c := attachLargeUnit(chunk{Read: last.Read, Sound: last.Sound, numericFragment: true}, string(chars[len(chars)-1]))
			tail := strings.TrimLeft(req.Number, "０〇")
			tailSound, err := r.integerFragmentSound(tail)
			if err != nil {
				return nil, err
			}
			_, tableTail := r.BySound[tailSound]
			tableTail = tableTail && strings.TrimLeft(prefix, "０〇") != ""
			if c.Sound == last.Sound && !tableTail {
				if req.Counter != "" {
					merged, err := r.withCounter(req, []chunk{c}, false)
					if err != nil {
						return nil, err
					}
					for i := range merged {
						merged[i].Groups = append(append([]Group(nil), groups[:len(groups)-1]...), merged[i].Groups...)
					}
					return merged, nil
				}
				readings[0].Groups[len(groups)-1] = r.accentOf(c)
				return readings, nil
			}
			// When the following unit assimilates, retain the leading zero
			// digit groups and evaluate the remaining numeric section. The
			// boundary has no written pause (００８京: ゼロゼロ / ハッケイ).
			zeros := req.Number[:len(req.Number)-len(tail)]
			zeroReadings, err := r.ReadDigits([]string{zeros}, false)
			if err != nil {
				return nil, err
			}
			partReq := req
			partReq.Number = tail
			merged, err := r.read(partReq, true)
			if err != nil {
				return nil, err
			}
			for i := range merged {
				merged[i].Groups = append(append([]Group(nil), zeroReadings[0].Groups...), merged[i].Groups...)
			}
			return merged, nil
		}
	}
	if len(chars) == 2 && (chars[0] == '〇' || chars[0] == '０') && strings.ContainsRune(string(bigUnits), chars[1]) {
		zero := r.digitWord(0)
		unit := string(chars[1])
		c := chunk{Read: zero.Read, Sound: zero.Sound, Key: unit, TkKey: unit, Unit: unit}
		if req.Counter != "" {
			return r.withCounter(req, []chunk{c}, false)
		}
		return []Reading{{Groups: []Group{r.accentOf(c)}}}, nil
	}
	// With Arabic digits and units, only the leading zeros form a digit
	// group. The nonzero suffix still contributes to its unit (０１十).
	if strings.HasPrefix(req.Number, "０") && strings.ContainsAny(req.Number, "十百千万億兆京垓") {
		tail := strings.TrimLeft(req.Number, "０")
		return r.readZeroParts(req, []string{req.Number[:len(req.Number)-len(tail)], tail})
	}
	// Preserve literal 零 and zero-prefixed runs at script and unit boundaries.
	// For example, ０〇一 reads ゼロ / ゼロイチ, and 〇一十 reads
	// ゼロイチ / ジュウ rather than evaluating the prefix as zero or one.
	parts := r.digitTypeRuns(req.Number)
	if (len(parts) > 1 && strings.ContainsRune(req.Number, '零') || unitBeforeZero(req.Number)) && strings.ContainsAny(req.Number, "十百千"+string(bigUnits)) {
		return r.readMixedZeroUnits(req)
	}
	if len(parts) > 1 {
		for _, part := range parts {
			if strings.ContainsRune(part, '零') || strings.HasPrefix(part, "〇") || strings.HasPrefix(part, "０") {
				return r.readZeroParts(req, parts)
			}
		}
	}
	if strings.HasPrefix(req.Number, "〇") || strings.HasPrefix(req.Number, "０") {
		for i, c := range req.Number {
			if _, digit := kanjiDigits[c]; !digit && i > 0 {
				return r.readZeroParts(req, []string{req.Number[:i], req.Number[i:]})
			}
		}
	}
	var chunks []chunk
	var scriptEnds map[int]bool
	if u, ok := r.Units[req.Number]; ok && (req.Number == "何" || req.Number == "数" || req.Number == "幾") {
		// an unknown quantity (ナン, スー, イク)
		chunks = []chunk{{Read: u.Read, Sound: u.Sound, Key: req.Number, TkKey: req.Number}}
	} else {
		// The original keeps a mixed-script numeral in one word but reads
		// each Japanese/full-width digit run as a separate numeric value.
		// Units attach to either kind; accent/counter processing still sees
		// the combined list of groups, as in the native merged word.
		parts := r.digitTypeRuns(req.Number)
		for partIndex, part := range parts {
			v, err := Value(part)
			if err != nil {
				return nil, err
			}
			if part == "零" {
				z := r.Digits[part]
				chunks = append(chunks, chunk{Read: z.Read, Sound: z.Sound, Key: part, TkKey: part})
			} else {
				chunks = append(chunks, r.numberChunks(v)...)
			}
			if partIndex < len(parts)-1 {
				if scriptEnds == nil {
					scriptEnds = map[int]bool{}
				}
				scriptEnds[len(chunks)-1] = true
				chunks[len(chunks)-1].fragmentBoundary = true
			}
		}
	}
	var groups []Group
	if req.Counter == "" {
		for i, c := range chunks {
			g := r.accentOf(c)
			if i < len(chunks)-1 {
				r.flatBeforeMore(c, &g)
			}
			groups = append(groups, g)
		}
		return addScriptPauses([]Reading{{Groups: groups}}, scriptEnds), nil
	}
	readings, err := r.withCounter(req, chunks, false)
	if err != nil {
		return nil, err
	}
	return addScriptPauses(readings, scriptEnds), nil
}

// readZeroParts retains digit-reading accent groups and their pauses. Only
// the final part receives the counter; alternative final readings are retained.
func (r *Rules) readZeroParts(req Request, parts []string) ([]Reading, error) {
	var prefix []Group
	for i, part := range parts {
		partReq := req
		partReq.Number = part
		if i < len(parts)-1 {
			partReq.Counter = ""
		}
		readings, err := r.read(partReq, true)
		if err != nil {
			return nil, err
		}
		if i == len(parts)-1 {
			for j := range readings {
				readings[j].Groups = append(append([]Group(nil), prefix...), readings[j].Groups...)
			}
			return readings, nil
		}
		prefix = append(prefix, readings[0].Groups...)
		prefix = append(prefix, Group{Read: "ｂ", Pause: true})
	}
	return nil, fmt.Errorf("empty number parts")
}

func fullWidthDigits(number string) bool {
	for _, c := range number {
		if c < '０' || c > '９' {
			return false
		}
	}
	return number != ""
}

// leadingZeroDigitRun excludes mixed scripts and unit numerals. Those retain
// their separate numeric groups rather than becoming one digit sequence.
func leadingZeroDigitRun(number string) bool {
	if strings.HasPrefix(number, "０") {
		return fullWidthDigits(number)
	}
	if !strings.HasPrefix(number, "〇") {
		return false
	}
	for _, c := range number {
		if !strings.ContainsRune("〇一二三四五六七八九", c) {
			return false
		}
	}
	return true
}

// readLeadingZero retains the digit groups and attaches a counter only to the
// last group. Digit groups have no numeric key, so they do not acquire a
// single digit's special reading or counter sound change (016本 is ロクホン).
// A digit fragment inside the general numeric path does retain its final key:
// 10f0a0's param11 differs between ０３本 and 一零０３本 (ホン vs ボン).
func (r *Rules) readLeadingZero(req Request, numeric bool) ([]Reading, error) {
	req.unkeyedDigits = !numeric
	readings, err := r.ReadDigits([]string{req.Number}, false)
	if numeric && leadingZeroDigitRun(req.Number) {
		// 10b010 keeps the leading zeros, evaluates the remaining integer,
		// and uses the short digit words. 1043f0's isolated digit sequence
		// instead lengthens 二/五 and leaves 012 as individual digits.
		tail := strings.TrimLeft(req.Number, "０〇")
		if tail != "" {
			zeros := req.Number[:len(req.Number)-len(tail)]
			readings, err = r.ReadDigits([]string{zeros}, false)
			if err != nil {
				return nil, err
			}
			v, e := Value(tail)
			if e != nil {
				return nil, e
			}
			groups := readings[0].Groups
			mergeDigit := v.IsInt64() && v.Int64() < 10 && utf8.RuneCountInString(zeros)%2 == 1
			if mergeDigit && req.Counter != "" {
				d := r.digitWord(int(v.Int64()))
				key := digitKanji[int(v.Int64())]
				c := chunk{Read: d.Read, Sound: d.Sound, Key: key, TkKey: key}
				counter, e := r.withCounter(req, []chunk{c}, false)
				if e != nil {
					return nil, e
				}
				// The native counter rebuild detaches a changed final
				// digit from its zero group: ゼロ / イッポン or ヨジ.
				if !strings.HasPrefix(counter[0].Groups[0].Sound, d.Sound) {
					mergeDigit = false
				}
			}
			if mergeDigit {
				d := r.digitWord(int(v.Int64()))
				last := groups[len(groups)-1]
				read, sound := last.Read+d.Read, last.Sound+d.Sound
				groups[len(groups)-1] = group(read, sound, 3, MoraCount(sound))
			} else {
				chunks := r.numberChunks(v)
				for i, c := range chunks {
					g := r.accentOf(c)
					if i < len(chunks)-1 {
						r.flatBeforeMore(c, &g)
					}
					groups = append(groups, g)
				}
			}
			readings[0].Groups = groups
		}
	}
	if err != nil || req.Counter == "" {
		return readings, err
	}
	groups := readings[0].Groups
	last := groups[len(groups)-1]
	tail := chunk{Read: last.Read, Sound: last.Sound}
	if numeric {
		digits := []rune(req.Number)
		lastDigit := digits[len(digits)-1]
		tail.Key = digitKanji[kanjiDigits[lastDigit]]
		if lastDigit == '零' {
			tail.Key = "零"
		}
		tail.TkKey = tail.Key
	}
	merged, err := r.withCounter(req, []chunk{tail}, false)
	if err != nil {
		return nil, err
	}
	if numeric && tail.Key != "" {
		base := r.digitWordByKey(tail.Key)
		if last.Sound != base.Sound && strings.HasSuffix(last.Sound, base.Sound) && !strings.HasPrefix(merged[0].Groups[0].Sound, last.Sound) {
			// A changed final digit is rebuilt separately from its paired
			// prefix: range 零九日 is レー / クニチ, not レークニチ.
			prefix := r.accentOf(chunk{Read: strings.TrimSuffix(last.Read, base.Read), Sound: strings.TrimSuffix(last.Sound, base.Sound)})
			counter, e := r.withCounter(req, []chunk{{Read: base.Read, Sound: base.Sound, Key: tail.Key, TkKey: tail.TkKey}}, false)
			if e != nil {
				return nil, e
			}
			readings[0].Groups = append(groups[:len(groups)-1], prefix)
			readings[0].Groups = append(readings[0].Groups, counter[0].Groups...)
			return readings, nil
		}
	}
	readings[0].Groups = append(groups[:len(groups)-1], merged[0].Groups...)
	return readings, nil
}

// Native mixed-script boundaries carry an s0 group and ｂ in the reading,
// even when the later frontend suppresses their audible pause.
func addScriptPauses(readings []Reading, ends map[int]bool) []Reading {
	if len(ends) == 0 {
		return readings
	}
	for i := range readings {
		var groups []Group
		for index, g := range readings[i].Groups {
			groups = append(groups, g)
			if ends[index] {
				groups = append(groups, Group{Read: "ｂ", Pause: true})
			}
		}
		readings[i].Groups = groups
	}
	return readings
}

// digitTypeRuns splits only at transitions between Jap and Eng digits.
// A unit does not reset the digit kind. Decimal inputs use the decimal reader.
func (r *Rules) digitTypeRuns(number string) []string {
	var parts []string
	start, kind := 0, ""
	for i, c := range number {
		next := r.Digits[string(c)].Type
		if next != "Jap" && next != "Eng" {
			continue
		}
		if kind != "" && kind != next {
			parts = append(parts, number[start:i])
			start = i
		}
		kind = next
	}
	return append(parts, number[start:])
}

// special turns a [特殊読み] entry into readings.
func (r *Rules) special(sp Special) ([]Reading, error) {
	var out []Reading
	for i, alt := range sp.Alternatives {
		var counts []int
		var falls []int
		for _, part := range strings.Split(alt.Accent, ",") {
			var fall, moras int
			if _, err := fmt.Sscanf(part, "%d-%d", &fall, &moras); err != nil {
				return nil, fmt.Errorf("special reading %q: %w", sp.Key, err)
			}
			counts = append(counts, moras)
			falls = append(falls, fall)
		}
		reads := splitMoras(alt.Read, counts)
		sounds := splitMoras(alt.Sound, counts)
		rd := Reading{Name: alt.Read, Connection: alt.Con}
		if i > 0 && alt.Read == sp.Alternatives[0].Read {
			rd.Name = alt.Read + "２"
		}
		for k := range counts {
			g := group(reads[k], sounds[k], falls[k], counts[k])
			g.fixed = true
			rd.Groups = append(rd.Groups, g)
		}
		out = append(out, rd)
	}
	return out, nil
}

// plain reports whether every alternative uses the neutral connection.
func (sp Special) plain() bool {
	for _, a := range sp.Alternatives {
		if a.Con != "*" {
			return false
		}
	}
	return len(sp.Alternatives) > 0
}

// IsNumberWord reports whether s is a single digit or unit of the rule file
// that the reader understands (full-width digits, kanji digits and units).
func (r *Rules) IsNumberWord(s string) bool {
	if utf8.RuneCountInString(s) != 1 {
		return false
	}
	rn, _ := utf8.DecodeRuneInString(s)
	if rn == '何' || rn == '数' || rn == '幾' {
		return true
	}
	if _, ok := kanjiDigits[rn]; ok {
		return true
	}
	if _, ok := smallUnits[rn]; ok {
		return true
	}
	for _, u := range bigUnits {
		if u == rn {
			return true
		}
	}
	return false
}

// IsConnector reports whether s is a numeral connector (decimal point, slash,
// hyphen, ...), which later patterns handle.
func (r *Rules) IsConnector(s string) bool {
	_, ok := r.Cons[s]
	return ok
}
