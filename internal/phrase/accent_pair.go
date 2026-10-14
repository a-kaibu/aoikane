package phrase

import "strings"

// pronEndsWith reports whether the base form ends with the given suffix (the
// native code compares the trailing bytes of the base form).
func baseEndsWith(m *Morph, suffix string) bool { return strings.HasSuffix(m.Base, suffix) }

// ae... helpers of the verb and adjective exceptions.

// aeb50 is 1aeb50: the base form ends with よい or 良い.
func aeb50(m *Morph) bool {
	return len(m.Base) >= 4 && (baseEndsWith(m, "よい") || baseEndsWith(m, "良い"))
}

// aee30 is 1aee30: a verb ending in 見る/みる that is not one of the listed
// compounds, with at least four moras.
func aee30(m *Morph) bool {
	rs := []rune(m.Base)
	if len(rs) < 3 || m.Moras < 4 {
		return false
	}
	tail2 := string(rs[len(rs)-2:])
	if tail2 != "見る" && tail2 != "みる" {
		return false
	}
	switch string(rs[len(rs)-3:]) {
	case "染みる", "滲みる", "しみる", "じみる", "凍みる", "沁みる", "侵みる", "試みる":
		return false
	}
	return true
}

// aece0 is 1aece0: a class-4 word whose third mora from the end is not ッ.
func aece0(m *Morph) bool {
	if m.POS&0xff000000 != 0x4000000 {
		return false
	}
	if (m.POS2&0xff00-0x200)&0xfffffbff != 0 {
		return false
	}
	rs := []rune(lastMoras(m, 3))
	if len(rs) == 0 {
		return true
	}
	return rs[0] != 'ッ'
}

// af190 is 1af190.
func af190(m *Morph, flag bool) bool {
	moras := m.Moras
	last := lastMoras(m, 1)
	iVar5 := moras
	if last == "レ" || last == "リャ" || last == "ッ" {
		iVar5 = moras - 1
	}
	if !flag || !aece0(m) {
		return false
	}
	isNai := len(m.Base) >= 4 && baseEndsWith(m, "ない")
	if isNai && m.Base != "少ない" {
		return false
	}
	if aeb50(m) {
		return false
	}
	if 3 < iVar5 {
		v := int32(0)
		if m.Down <= m.Moras {
			v = m.Down
		}
		if m.Moras < v {
			return false
		}
		sub := moraRange(m, int(v)-1, 1)
		if sub != "シ" && sub != "キ" {
			return false
		}
	}
	return true
}

// adb00 is 1adb00: connections of a prefix and a doubled word. It returns a
// negative value when no decision is made.
func adb00(cur, next *Morph, accum, fall int32) int32 {
	if fall == 0 && cur.POS&0xff000000 == 0x5000000 && next.POS&0xff000000 == 0x9000000 {
		return accum
	}
	if cur.POS&0xff000000 == 0x5000000 {
		if next.Surface == "だ" || next.Surface == "な" || next.Surface == "に" {
			rs := []rune(cur.Surface)
			if len(rs)%2 == 0 && len(rs) > 0 && string(rs[:len(rs)/2]) == string(rs[len(rs)/2:]) {
				return 0
			}
		}
	}
	lVar1 := next.Next
	class := cur.POS & 0xff000000
	if (class == 0x3000000 || class == 0x4000000 || class == 0x9000000) && next.POS2 == 0x300 && lVar1 != nil &&
		(lVar1.p16b950() || lVar1.POS&0xff000000 == 0x8000000 || lVar1.p16e680() ||
			((lVar1.POS&0xff000000)+0xfd000000)&0xfeffffff == 0 && lVar1.POS2&0xff00 == 0x300 &&
				(lVar1.Next == nil || lVar1.Next.p16e680())) {
		return accum + 1
	}
	return -1
}

// add00 is 1add00: the rule that connects the next word to the current one.
// It returns a negative value when no rule applies.
func add00(flag bool, cur, next *Morph, accum, fall int32, rise *int32) int32 {
	// The preceding word's own unconditional rule (P1..P4, prefixes).
	if c := cur.SelectedCandidate(); c != nil {
		rule, _, _ := ParseRule(c.Accent, c.Connection, 0, next.Moras)
		if rule < 4 {
			p := next.Down
			n := next.Moras
			var flat bool
			if n < p {
				p = 0
				flat = true
			} else if p == -1 || p == 0 || 1 < p && p == n {
				flat = true
			} else if 1 < p && p < n {
				d := int32(0)
				if s := []rune(lastMoras(next, 1)); len(s) != 0 && isSpecialMora(s[0]) {
					d = 1
				}
				flat = p == n-d
			}
			if cur.POS == 0x2000000 && next.p16b750() {
				p = next.Moras
				if p < next.Down || next.Down == 0 || p == 4 {
					p = 0
				}
				flat = true
			}
			switch rule {
			case 0:
				fall = 0
				if flat {
					return fall
				}
			case 1:
				if flat {
					return accum + 1
				}
			case 3:
				if flat {
					return accum + 1
				}
				return fall
			default:
				return fall
			}
			return p + accum
		}
	}
	classes := ac760(cur)
	af190(cur, flag) // evaluated by the original, result unused
	nextCand := next.SelectedCandidate()
	class := 0
	for i := 0; i < 3; i++ {
		if classes[i] == 0 {
			break
		}
		if nextCand == nil {
			continue
		}
		if rule, _, _ := ParseRule(nextCand.Accent, nextCand.Connection, classes[i], -1); rule != RuleAbsent {
			class = classes[i]
			break
		}
	}
	downOrZero := func() int {
		if next.Down <= next.Moras {
			return int(next.Down)
		}
		return 0
	}
	if nextCand != nil {
		rule, first, second := ParseRule(nextCand.Accent, nextCand.Connection, class, cur.Moras)
		switch {
		case rule >= 8 && rule < 14:
			if second == unset {
				second = downOrZero()
			}
			s := int32(second)
			switch rule {
			case 9:
				if fall == 0 {
					return s + accum
				}
			case 10:
				if fall == 0 {
					return 0
				}
				return s + accum
			case 11:
				return s + accum
			case 12:
				return 0
			case 13:
				if fall != 0 && fall != 1 && fall == accum {
					fall = 0
				}
			}
			return fall
		case rule >= 17 && rule <= 24:
			if second == unset {
				second = downOrZero()
			}
			n := next.Moras
			use := rule
			off := int32(second)
			if p := next.POS; (p == 0x3020000 || p == 0x3010000 || p == 0x9000000) && next.POS2-0x500 < 5 {
				if cur.POS3 == 0x40b || next.POS2 == 0x501 || next.Base == "なはる" {
					off = n - 2
				} else {
					off = n - 1
				}
				use = 0x14
			}
			return ae3a0(use, fall, accum, rise, off, n, int32(first), cur)
		}
		// default (class 0) rule of the next word
		rule, _, second = ParseRule(nextCand.Accent, nextCand.Connection, 0, cur.Moras)
		if rule >= 4 && rule < 8 {
			p5 := fall
			if cur.p16b750() {
				p5 = cur.Moras
				if p5 < cur.Down || cur.Down == 0 || p5 == 4 {
					p5 = 0
				}
			}
			off := int32(second)
			if second == unset {
				off = next.Down
				if next.Moras < off {
					off = 0
				}
			}
			r := ad920(int32(rule), p5, accum, off, cur, next)
			if !af190(cur, flag) || p5 < 1 {
				return r
			}
			if p5 < accum {
				if r == p5 {
					return r
				}
				if 1 < r {
					return r - 1
				}
			}
			return r
		}
		if rule == 0x10 {
			p := next.Down
			if next.Moras < p {
				return 0
			}
			if p != 0 {
				return p + accum
			}
			return 0
		}
	}
	return -1
}
