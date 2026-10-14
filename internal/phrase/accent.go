package phrase

import "strings"

// Accent combination within one prosodic word. Every function is named after
// the native address it reconstructs. Positions are mora counts; a rise or fall
// of 0 means "none".

const unset = -9999 // 0xffffd8f1

// charAtMora returns the character at position i of a pronunciation (the
// native code indexes by two-byte unit), or "" past the end.
func charAtIndex(s string, i int) string {
	if i < 0 {
		return ""
	}
	rs := []rune(s)
	if i >= len(rs) {
		return ""
	}
	return string(rs[i])
}

// ac760 is 1ac760: the connection classes of the preceding word that are tried
// in order (1 noun, 2 verb, 3 adjective, 4 special auxiliary, 5 relative clause).
func ac760(m *Morph) [3]int {
	switch m.POS & 0xff000000 {
	case 0x4000000:
		return [3]int{3, 1, 0}
	case 0x3000000:
		return [3]int{2, 1, 0}
	case 0x9000000:
		if m.p16b920() {
			return [3]int{3, 2, 0}
		}
		if m.POS3-0xc02 > 2 {
			return [3]int{2, 1, 0}
		}
		return [3]int{4, 2, 1}
	}
	if m.POS3 == 0 {
		return [3]int{1, 0, 0}
	}
	return [3]int{2, 1, 0}
}

// ad920 is 1ad920: the fall position for the C rules (rule 4..7).
func ad920(rule, fall, accum, offset int32, cur, next *Morph) int32 {
	switch rule {
	case 4:
		return accum + offset
	case 5:
		return accum + 1
	case 6:
	default:
		if rule != 7 {
			return fall
		}
		return 0
	}
	if (next.POS+0xfef6ff00)&0xfffffeff == 0 {
		return accum
	}
	if cur.POS+0xfefefd00 < 2 {
		return accum
	}
	if cur.Moras < 2 {
		return accum
	}
	rs := []rune(lastMoras(cur, 2))
	if len(rs) == 2 {
		if isSpecialMora(rs[0]) {
			if isSpecialMora(rs[1]) {
				accum--
			}
		} else if isVowelKana(rs[1]) || isSpecialMora(rs[1]) {
			accum--
		}
	} else if len(rs) == 3 && matchesSecondPrefix(string(rs[1:])) {
		if !isSpecialMora(rs[0]) && isVowelKana(rs[2]) {
			accum--
		} else if isSpecialMora(rs[2]) {
			accum--
		}
	}
	return accum
}

// secondPrefixes is the runtime list consulted by 196400.
var secondPrefixes = []string{
	"ァ", "ィ", "ゥ", "ェ", "ォ", "ャ", "ュ", "ョ", "ヮ", "゜",
	"クァ", "クィ", "クェ", "クォ", "キャ", "キュ", "キェ", "キョ",
	"グァ", "グィ", "グェ", "グォ", "ギャ", "ギュ", "ギェ", "ギョ",
	"スァ", "スィ", "スェ", "スォ", "シャ", "シュ", "シェ", "ショ",
}

// matchesSecondPrefix is 196400: the string starts with one of the listed
// prefixes.
func matchesSecondPrefix(s string) bool {
	for _, p := range secondPrefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// ae3a0 is 1ae3a0: the K rules (0x11..0x18) and the generic fall rule 0x14.
// rise is updated in place; the result is the new fall.
func ae3a0(rule int, fall, accum int32, rise *int32, offset, nextMoras, first int32, cur *Morph) int32 {
	moras := cur.Moras
	p2, p3, p5, p6, p7 := fall, accum, offset, nextMoras, first
	switch rule {
	case 0x11:
		a := func() {
			if 1 < *rise && *rise < p3+p6 {
				*rise = p3 + p6
			}
		}
		if p3 < 2 || p3 != *rise {
			if p2 == 0 {
				a()
			}
		} else {
			if p2 != 0 {
				return p2
			}
			if p7 == 1 {
				*rise = p3 + 1
			}
			a()
		}
		if p3 == 1 && *rise == 0 && 1 < p6+1 {
			*rise = p6 + 1
		}
	case 0x12:
		x := false
		if *rise == 1 && p2 == 0 && p3 == 2 && cur.POS&0xff000000 == 0x3000000 && cur.POS2 == 0x602 {
			x = charAtIndex(cur.Pronunciation(), int(moras)-1) == "ン"
		}
		iVar10 := p2
		if x {
			iVar10 = 1
		}
		if cur.POS&0xff000000 == 0x3000000 && p5 == -1 {
			if n := cur.Next; n != nil && n.Next != nil {
				y := n.POS&0xff000000 == 0x8000000 && n.Pronunciation() == "バ"
				if y && *rise == moras && cur.POS3&0xff00 == 0x400 && moras == 2 {
					p5 = 0
				}
			}
		}
		if iVar10 == 0 {
			iVar10 = p3 + p5
		}
		iVar5 := iVar10
		if p5 < 0 && 0 <= moras+p5 {
			m := moras - p3 + iVar10
			z := false
			if m >= 1 {
				z = charAtIndex(cur.Pronunciation(), int(m)-1) == "ッ"
			}
			if z {
				iVar10--
				if p3 == *rise && iVar10 < 2 {
					*rise = 1
				}
			}
			z = false
			if *rise == p3 && iVar10 == p3-1 {
				z = charAtIndex(cur.Pronunciation(), int(m)) == "ー"
			}
			iVar5 = iVar10
			if z {
				iVar5 = iVar10 + 2
			}
			if iVar5 < 1 {
				iVar5 = 1
			}
		}
		r := *rise
		if r != 1 || p3 != 2 {
			if r != p3 {
				return iVar5
			}
			if r <= iVar5 {
				return iVar5
			}
			*rise = iVar5
			return iVar5
		}
		if iVar5 != 1 {
			return iVar5
		}
		if p5 != 0 {
			return 1
		}
		if cur.POS2 != 0x600 && cur.POS2 != 0x403 {
			return 1
		}
		return 2
	case 0x13:
		if p2 == 0 {
			return 0
		}
		p2 = p5 + p3
	case 0x14:
		p2 = p5 + p3
	case 0x15:
		p2 = 0
		b := func() {
			if p3 <= *rise && 1 < p3 && 1 < p7 && p5 == 0 {
				*rise = p7 + p3
			}
		}
		if p3 < 2 {
			if p3 == 1 && *rise == 0 && 1 < p6+1 {
				*rise = p6 + 1
			}
			b()
		} else if p3 <= *rise {
			*rise = p6 + p3
			b()
		}
		if cur.Base == "たげる" {
			*rise = 1
		}
	case 0x16:
		p2 = 0
		if 1 < p3 && p3 <= *rise {
			*rise = p6 + p3
		}
	case 0x17:
		if p2 == 0 {
			p2 = p3 + p5
		}
		if p3 == 1 {
			*rise = 1
		}
		if p5 < 1 && -p5 < moras && p5 == -1 && (cur.POS+0xfd000000)&0xfffcffff == 0 && cur.POS != 0x3030000 {
			if *rise == 1 {
				if cur.POS3&0xff00 == 0x400 {
					if cur.Moras == 3 {
						p2--
					}
				} else if cur.Moras == 1 {
					p2++
				}
			}
			if p3 <= *rise {
				if cur.POS3&0xff00 == 0x400 {
					if cur.Moras == 2 {
						p2 = p3 + 1
						*rise = p2
					}
				} else if cur.Moras < 3 {
					p2++
				}
			}
		}
		if p2 < 1 {
			p2 = 1
		}
	case 0x18:
		if p2 != 0 {
			if p2 > 0 {
				return p2
			}
			return 1
		}
		if 2 < p3 || *rise == 0 && p3 == 2 {
			if 0 < p5+p3 {
				return p5 + p3
			}
			return 1
		}
		if *rise != p3 || p3 != 2 {
			*rise = 1
			return 1
		}
		return 2
	}
	return p2
}
