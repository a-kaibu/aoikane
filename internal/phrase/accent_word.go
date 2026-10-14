package phrase

import "strings"

// kindInfo is the per-boundary-kind table at 180391310: the strength of the
// boundary and its class (2 marks a strong boundary).
var kindInfo = [...]struct {
	Strength float32
	Class    int32
}{
	{0, 0}, {0, 0}, {0.5, 2}, {0.21, 1}, {0.35, 2}, {0, 0}, {1.0, 2}, {0, 0}, {0, 0}, {0.35, 2},
	{0, 0}, {0.22, 2}, {0, 0}, {0.33, 2}, {0.23, 2}, {0.5, 2}, {-0.1, 0}, {-0.1, 0}, {-0.1, 0}, {-0.1, 2},
	{-0.1, 2}, {0, 0}, {0.3, 2}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0.23, 1}, {0.33, 2},
	{0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0},
}

func kindStrength(kind int32) float32 {
	if kind < 0 || int(kind) >= len(kindInfo) {
		return 0
	}
	return kindInfo[kind].Strength
}

func kindClass(kind int32) int32 {
	if kind < 0 || int(kind) >= len(kindInfo) {
		return 0
	}
	return kindInfo[kind].Class
}

// ad010 is 1ad010: it advances the rise and fall of a prosodic word by one
// morph. accum is the number of moras up to and including cur; next is nil for
// the last morph of the word.
func (d *Document) ad010(rise, fall *int32, accum int32, cur, next *Morph) {
	iVar22 := *rise
	iVar19 := *fall
	local94 := iVar22
	if cur.silent() {
		return
	}
	bVar7, bVar8 := false, false
	if 1 < iVar19 {
		if cur.POS&0xff000000 == 0x3000000 && cur.p16b900() && (cur.POS2&0xff00-0x400)&0xfffffdff == 0 {
			bVar7 = true
			if p := cur.Prev; p != nil && p.Prev != nil && p.Prev.POS == 0x1040000 && p.POS3 == 0x200 && cur.POS&0xff000000 == 0x3000000 {
				if cur.p16b980() {
					bVar7 = false
				}
			}
		}
		if d.Flag0 && aece0(cur) && af190(cur, d.Flag0) {
			bVar8 = true
		}
	}
	verbStem := cur.p16b750()
	if d.Mode == 1 {
		match := func(pron string) bool { return cur.Surface == "" && cur.Pronunciation() == pron }
		if match("アカン") {
			iVar22 = 1
		}
		if match("エエ") {
			iVar22 = 2
		}
		local94 = iVar22
		if next == nil && accum == iVar22 && iVar19 == 0 &&
			(cur.POS == 0x1000000 || cur.POS == 0x1010300 || cur.POS == 0x1020000) {
			if n := cur.Next; n != nil && n.Next != nil && (n.POS == 0x3000000 || n.POS == 0x4000000) {
				v := n.Moras
				if n.Up <= n.Moras {
					v = n.Up
				}
				if v == 1 {
					local94 = 0
				}
			}
		}
		iVar22 = local94
		pos := cur.POS
		if (pos == 0x3000000 || pos == 0x3010000 || pos == 0x3020000 || pos == 0x9000000) && cur.POS2-0x500 < 5 && cur.PWord != nil {
			iVar2 := cur.Moras
			if cur.POS3 != 0x40b || cur.POS2 != 0x501 || iVar2 < 3 {
				if cur.Base != "なはる" {
					if (cur.Up < iVar2 || iVar2 != 2) && 1 < iVar2 {
						iVar19 = iVar2 - 1
					}
					goto adjust
				}
			}
			iVar19 = iVar2 - 2
		}
	adjust:
		if pos == 0x4000000 {
			iVar11 := cur.dictMoras()
			iVar10 := cur.Moras
			iVar2 := iVar10
			if cur.Up <= iVar10 {
				iVar2 = cur.Up
			}
			iVar18 := int32(0)
			if cur.Down <= iVar10 {
				iVar18 = cur.Down
			}
			uVar3 := cur.POS2
			out := int32(0)
			if uVar3 != 0x700 {
				if uVar3 == 0x603 {
					out = iVar11 - 1
				} else {
					out = iVar19
					if uVar3 == 0x601 || uVar3 == 0x202 {
						if iVar11 < 3 {
							goto standard
						}
						if iVar2 == 1 && next == nil {
							out = iVar18 + 1
						}
					}
				}
			}
			iVar19 = out
			if iVar11 == 3 && iVar2 == 1 {
				if uVar3 == 0x603 || uVar3 == 0x601 || uVar3&0xff00 == 0x200 {
					local94, iVar22, iVar19 = 2, 2, 2
				} else if uVar3&0xff00 == 0x600 {
					if iVar18 == 0 {
						local94, iVar22 = accum, accum
					} else {
						iVar22, local94 = iVar19, iVar19
						if iVar18 == 1 {
							local94, iVar22 = accum, accum
							iVar19 = 0
						}
					}
				}
			}
		}
	}
standard:
	if d.Mode == 0 {
		endsDeru := len(cur.Base) >= 4 && baseEndsWith(cur, "出る")
		if !endsDeru && !aee30(cur) && (bVar7 || bVar8) && (cur.POS+0xfcff0000)&0xfeffffff != 0 {
			if cur.Base != "訴える" || iVar19 != 3 {
				if cur.Base != "してくれる" && cur.Base != "てくれる" && cur.Base != "てる" && cur.Base != "してみる" {
					iVar19--
				}
			}
		}
	}
	if next == nil {
		*fall, *rise = iVar19, iVar22
		return
	}
	if next.silent() {
		if verbStem && next.POS != 0xb020000 {
			*fall = 0
			return
		}
		*fall = iVar19
		return
	}
	iVar22 = accum
	var iVar10 int32 = -1
	if d.Mode != 0 || func() bool { iVar10 = adb00(cur, next, accum, iVar19); return iVar10 < 0 }() {
		iVar10 = add00(d.Flag0, cur, next, accum, iVar19, &local94)
		if iVar10 >= 0 {
			*fall, *rise = iVar10, local94
			return
		}
		iVar10 = next.Moras
		if next.POS&0xff000000 == 0x3000000 && cur.p16b750() && next.p16b7f0() && next.Next != nil {
			if next.p16b750() && next.Next.POS&0xff000000 == 0x8000000 {
				*fall = 0
				return
			}
			iVar2 := next.Down
			if next.Moras < iVar2 || iVar2 == 0 {
				iVar2 = iVar10 - 1
			}
			iVar10 = iVar2 + iVar22
			if iVar10 >= 0 {
				*fall = iVar10
				return
			}
		}
		iVar10 = 0
		if next.Down <= next.Moras {
			iVar10 = next.Down
		}
		if iVar19 == 0 {
			iVar19 = iVar22 + iVar10
		}
		*fall = iVar19
		return
	}
	*fall = iVar10
}

// dictMoras is 168680: the third number of the selected accent text.
func (m *Morph) dictMoras() int32 {
	c := m.SelectedCandidate()
	text := ""
	if c != nil {
		text = c.Accent
	}
	i := strings.LastIndexByte(text, '-')
	if i < 0 {
		return int32(atoi(""))
	}
	return int32(atoi(text[i+1:]))
}
