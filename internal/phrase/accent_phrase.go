package phrase

// ac9b0 is 1ac9b0: it computes the accent of a prosodic word (rise, fall and
// mora count) from its morphs and the previous prosodic word.
func (d *Document) ac9b0(prev, pw *PWord) {
	end := pw.endMorph()
	v := pw.First
	for v != nil && v != end {
		if v.Moras > 0 {
			break
		}
		v = v.Next
		if v == nil {
			break
		}
	}
	moras0 := v.Moras
	iVar15 := moras0
	if v.Up <= moras0 {
		iVar15 = v.Up
	}
	iVar12 := v.Down
	iVar5 := int32(0)
	if iVar12 <= moras0 {
		iVar5 = iVar12
	}
	iVar14 := iVar5
	rise := iVar15
	if prev != nil {
		if pm := v.Prev; pm != nil && ((pm.POS2&0xff00-0x100)&0xfffffdff == 0 || pm.POS&0xff000000 == 0x6000000 || pm.POS == 0x8080000) {
			if c := v.SelectedCandidate(); c != nil {
				rule, _, second := ParseRule(c.Accent, c.Connection, 5, pm.Moras)
				if rule == RuleA1 && (prev.Down == 0 || prev.Down == prev.Moras) {
					iVar14 = int32(second)
				}
			}
		}
	}
	fall := iVar14
	accum := int32(0)
	for m := v; m != nil && m != end; {
		if m == end.Prev {
			break
		}
		accum += m.Moras
		next := m.Next
		if next == nil || next.Next == nil {
			next = nil
		}
		d.ad010(&rise, &fall, accum, m, next)
		m = m.Next
		if m == nil || m.Next == nil {
			goto done
		}
		v = m
	}
	{
		// the last morph of the word
		m := end.Prev
		accum += m.Moras
		d.ad010(&rise, &fall, accum, m, nil)
	}
done:
	iVar15 = rise
	iVar12 = fall
	iVar4 := accum
	var out int32
	if d.Mode != 1 {
		for iVar12 >= 2 && firstSpecial(moraAt(pw, int(iVar12)-1)) {
			iVar12--
		}
		switch {
		case iVar4 == 1 && iVar12 != 1:
			out = 0
		case iVar4 == 1:
			out = 1
		case iVar12 != 1:
			out = 2
		default:
			out = 1
		}
	} else {
		if pw.First.POS == 0x1060000 {
			for iVar12 >= 2 && firstSpecial(moraAt(pw, int(iVar12)-1)) {
				iVar12--
			}
			if iVar12 < iVar15 {
				iVar15 = 1
				if iVar12 != 1 {
					iVar15 = 2
				}
			}
		}
		for iVar12 >= 2 && moraAt(pw, int(iVar12)-1) == "ッ" {
			iVar12--
		}
		out = iVar15
		if pw.First != nil && pw.First.POS != 0x1060000 {
			if iVar15 == 0 {
				if iVar12 != 0 {
					out = iVar12
				}
			} else if iVar15 >= 2 {
				out = iVar4
				if iVar12 != 0 {
					out = iVar15
					if iVar12 >= 1 {
						out = iVar12
					}
				}
			}
		}
		if out == 1 && prev != nil && prev.Down == 0 && prev.Up == prev.Moras {
			last := prev.Next.First.Prev
			s := kindStrength(last.Kind)
			for m := last; m.Moras < 1 && m.silent() && m.Prev != nil; m = m.Prev {
				t := kindStrength(m.Kind)
				if m.Base == "、" {
					t = 0.2
				}
				if t <= s {
					t = s
				}
				s = t
			}
			if s <= 0.01 {
				pw.Prev.Up = 0
			}
		}
	}
	pw.Up, pw.Down, pw.Moras = out, iVar12, iVar4
}
