package numeral

import (
	"fmt"
	"strings"
)

// sokuonDigits lists, per first Tn letter of a counter, the digits (and tens
// and hundreds) whose reading ends in ッ before it (イッ, ロッ, ハッ, ジュッ,
// ヒャッ). The Tk rows only say that a change is possible; which digits take it
// was read off the engine's output.
var sokuonDigits = map[byte]string{
	'G': "一六八十百",
	'H': "一八十",
	'L': "一六十百",
	'I': "六十百",
	'J': "十",
	'M': "一十",
	'K': "十",
}

// altDigit is the reading of a digit used before counters of a class: ヨ for
// 四 before 時 年 人 円, シチ for 七, ク for 九 (読み分け).
var altDigit = map[byte]map[string]Word{
	'B': {"四": {Read: "ヨ", Sound: "ヨ"}},
	'C': {"四": {Read: "ヨ", Sound: "ヨ"}, "七": {Read: "シチ", Sound: "シチ"}, "九": {Read: "ク", Sound: "ク"}},
	'E': {"四": {Read: "シ", Sound: "シ"}, "七": {Read: "シチ", Sound: "シチ"}, "九": {Read: "ク", Sound: "ク"}},
	'P': {"九": {Read: "ク", Sound: "ク"}},
	'N': {"四": {Read: "シ", Sound: "シ"}},
	'M': {"七": {Read: "シチ", Sound: "シチ"}},
}

func tkCode(row []TkCode, letter byte) int {
	i := int(letter) - 'A'
	if i < 0 || i >= len(row) {
		return 0
	}
	return row[i].Value
}

func tjCode(row string, letter byte) int {
	i := int(letter) - 'A'
	if i < 0 || i >= len(row) {
		return 0
	}
	return int(row[i] - '0')
}

var voiced = map[rune]rune{'カ': 'ガ', 'キ': 'ギ', 'ク': 'グ', 'ケ': 'ゲ', 'コ': 'ゴ', 'サ': 'ザ', 'シ': 'ジ', 'ス': 'ズ', 'セ': 'ゼ', 'ソ': 'ゾ',
	'タ': 'ダ', 'チ': 'ヂ', 'ツ': 'ヅ', 'テ': 'デ', 'ト': 'ド', 'ハ': 'バ', 'ヒ': 'ビ', 'フ': 'ブ', 'ヘ': 'ベ', 'ホ': 'ボ', 'ワ': 'バ'}
var semiVoiced = map[rune]rune{'ハ': 'パ', 'ヒ': 'ピ', 'フ': 'プ', 'ヘ': 'ペ', 'ホ': 'ポ', 'ワ': 'パ'}

func changeFirst(s string, table map[rune]rune) string {
	rs := []rune(s)
	if len(rs) > 0 {
		if v, ok := table[rs[0]]; ok {
			rs[0] = v
		}
	}
	return string(rs)
}

// geminate replaces the last mora of a number reading by ッ when it ends in
// チ, ク or ウ (イチ イッ, ロク ロッ, ジュウ ジュッ, ヒャク ヒャッ).
func geminate(s string) string {
	rs := []rune(s)
	if n := len(rs); n > 0 {
		switch rs[n-1] {
		case 'チ', 'ク', 'ウ', 'ー':
			rs[n-1] = 'ッ'
		}
	}
	return string(rs)
}

// chunkWord is the reading of a chunk without a counter: the table word of its
// pronunciation when there is one, otherwise the pieces, and the group unit
// (万, 億, ...) attached.
func (r *Rules) chunkWord(c chunk) (read, sound string) {
	read, sound = c.Read, c.Sound
	if w, ok := r.BySound[sound]; ok {
		read = w.Read
	}
	if c.Unit != "" {
		u := r.Units[c.Unit]
		read, sound = read+u.Read, sound+u.Sound
	}
	return
}

// withCounter attaches the counter to the last chunk of the number.
func (r *Rules) withCounter(req Request, chunks []chunk, decimalTail bool) ([]Reading, error) {
	ctr, ok := r.Counters[req.Counter]
	if req.Counter == "対比" {
		// the ratio of ３：２ reads タイ without sound change (イチタイ)
		ctr, ok = Counter{Word: Word{Read: "タイ", Sound: "タイ"}, Tn: "JAA", Tl: "C1"}, true
	}
	if req.Counter == "対" {
		// 対 acts as a counter with its own accent (サンタイ)
		ctr, ok = Counter{Word: Word{Read: "タイ", Sound: "タイ"}, Tn: "HAA", Tl: "C1"}, true
	}
	if !ok || len(ctr.Tn) != 3 {
		return nil, fmt.Errorf("unknown counter %q", req.Counter)
	}
	last := chunks[len(chunks)-1]
	tn := ctr.Tn
	rowTj := r.Tj[last.Key]
	c1 := tjCode(rowTj, tn[1])
	var c2 int
	switch {
	case decimalTail && last.Unit == "":
		// the last digit of a decimal fraction falls after itself
	case last.Key == "千" && last.TkKey == last.Key && !last.numericFragment:
		// a bare 千 always falls after the number
	case last.TkKey != last.Key && last.Unit == "":
		// a digit with a unit (二十, 三百): the unit's row, where the entries
		// marked with '!' only hold for the bare unit (十, 百)
		if row := r.Tk[last.Key]; int(tn[2])-'A' < len(row) && int(tn[2]) >= 'A' && !row[int(tn[2])-'A'].Bang {
			c2 = row[int(tn[2])-'A'].Value
		}
	default:
		c2 = tkCode(r.Tk[last.TkKey], tn[2])
	}

	read, sound := r.chunkWord(last)
	if req.compoundDigits {
		c2 = 0
		if w := r.BySound[sound]; len(w.Flag) < 5 || w.Flag[4] != '1' {
			c1 = 0
		}
		if ctr.Tl == "C3" {
			ctr.Tl = "C4"
		}
	}
	// Large-unit groups longer than four morae bypass the short-number
	// Tk adjustment. C3 then falls before the group's final mora.
	longUnit := last.Unit != "" && !last.tableStem && !req.pairedUnit && MoraCount(sound) > 4
	if longUnit {
		c2 = 0
	}
	// counters of English words (ドア, ...) read the digits 1-10 in English
	english := false
	if tn[0] == 'Q' && len(chunks) == 1 {
		for fw, d := range r.Digits {
			if d.Type == "Eng" && d.Jap == last.Key && last.Unit == "" && last.Key == last.TkKey && d.Eng != "" {
				read, sound = d.Eng, d.Eng
				english = true
				_ = fw
			}
		}
		if last.Key == "十" && last.TkKey == "十" {
			read, sound, english = "テン", "テン", true
		}
	}
	// alternative digit reading (ヨ, シチ, ク) when the chunk ends in the digit
	if last.Unit == "" && last.Key == last.TkKey {
		if alt, ok := altDigit[tn[0]][last.Key]; ok {
			base := r.digitWordByKey(last.Key)
			suffix := base.Read
			if !strings.HasSuffix(read, suffix) && strings.HasSuffix(read, base.Sound) {
				suffix = base.Sound
			}
			if strings.HasSuffix(read, suffix) {
				read = strings.TrimSuffix(read, suffix) + alt.Read
				sound = strings.TrimSuffix(sound, base.Sound) + alt.Sound
			}
		}
	}
	cread, csound := ctr.Read, ctr.Sound
	switch c1 {
	case 1:
		cread, csound = changeFirst(cread, voiced), changeFirst(csound, voiced)
	case 2:
		cread, csound = changeFirst(cread, semiVoiced), changeFirst(csound, semiVoiced)
	}
	sok := c1 == 2 || (last.Key != "" && strings.Contains(sokuonDigits[tn[0]], last.Key))
	if !sok && strings.ContainsRune("パピプペポ", firstRune(csound)) && tn[0] != 'K' && last.Key != "" && strings.Contains("一六八十", last.Key) {
		sok = true
	}
	if sok {
		read, sound = geminate(read), geminate(sound)
	}
	dm := MoraCount(sound)
	moras := dm + MoraCount(csound)
	var fall int
	switch ctr.Tl {
	case "C1", "C2":
		switch {
		case c2 == 1:
			// Tk=1 keeps the whole counter group flat, including C1/C2.
			// Large units before 個年 carry this class (ケイカネン).
			fall = 0
		case c2 == 3 && ctr.Tl == "C2":
			fall = moras
		case c2 == 2 && !english && tn[2] != '\\':
			fall = dm + 1
		// Native 10f0a0's zero-Tk C2 branch adds one independently
		// of the counter's dictionary accent; only C1 uses that accent.
		case req.CounterFall > 0 && ctr.Tl == "C1":
			fall = dm + req.CounterFall
		default:
			fall = dm + 1
		}
	case "C3":
		if longUnit {
			fall = dm - 1
			break
		}
		switch c2 {
		case 0:
			fall = dm
		case 1:
			fall = 0
		case 2:
			fall = dm + 1
		default:
			fall = moras
		}
	default:
		fall = 0
	}
	var groups []Group
	for _, c := range chunks[:len(chunks)-1] {
		g := r.accentOf(c)
		r.flatBeforeMore(c, &g)
		groups = append(groups, g)
	}
	merged := group(read+cread, sound+csound, fall, moras)
	if w, ok := r.BySound[sound+csound]; ok && !req.unkeyedDigits && len(w.Flag) > 2 && w.Flag[2] == '1' {
		merged = group(w.Read, sound+csound, w.Fall, w.Moras)
	}
	groups = append(groups, merged)
	return []Reading{{Groups: groups}}, nil
}

func firstRune(s string) rune {
	for _, r := range s {
		return r
	}
	return 0
}

func (r *Rules) digitWordByKey(key string) Digit {
	if key == "零" {
		return r.Digits[key]
	}
	for i, k := range digitKanji {
		if k == key {
			return r.digitWord(i)
		}
	}
	return Digit{}
}
