package phrase

// tagMorph reports the tag state used by the native conditions around 1afec0
// and 179600. Tag text is not supported: the loader rejects morphs that carry
// it, so these helpers only distinguish silent morphs.

// afec0 is 1afec0: every morph in [from, to) is silent.
func afec0(from, to *Morph) bool {
	for m := from; m != to; m = m.Next {
		if m == nil {
			return true
		}
		if !(m.Moras < 1 && m.TagPtr == 0 && uint32(m.Kind)-0x18 > 1) {
			if m.Kind == 0x18 {
				return false // tag-text check (179440) is not supported
			}
			if m.Kind != 0x19 || m.TagLen != 0 {
				return false
			}
		}
	}
	return true
}

// ac870 is 1ac870: whether the connection of two prosodic words keeps the
// accent of the earlier one when the S3 rule asks for a merge.
func ac870(prev, cur *PWord) bool {
	var last *Morph
	if prev.Next != nil && prev.Next.First != nil {
		last = prev.Next.First.Prev
	}
	first := cur.First
	classes := ac760(last)
	rule := RuleAbsent
	for i := 0; i < 3; i++ {
		c := first.SelectedCandidate()
		if c == nil {
			rule = RuleAbsent
			continue
		}
		rule, _, _ = ParseRule(c.Accent, c.Connection, classes[i], last.Moras)
		if rule != RuleAbsent {
			break
		}
	}
	if uint32(rule)-0xb > 1 {
		if prev.Down < 1 || prev.Moras <= prev.Down {
			if cur.Down > 0 {
				return cur.Down < cur.Moras
			}
			return false
		}
		return true
	}
	return false
}

// afc70 is 1afc70: the following morph inherits the classification and labels.
func afc70(prev, cur *Morph) {
	cur.Kind = prev.Kind
	cur.Annotation = prev.Annotation
	cur.Label170 = prev.Label170
}

// rankOf is the position of a kind in the table at 180392ee0, 0 if absent.
func rankOf(kind int32) int {
	for i, k := range [...]int32{16, 17, 28, 29, 14, 13, 30} {
		if k == kind {
			return i
		}
	}
	return 0
}

func silentTagFree(m *Morph) bool { return m.Moras < 1 && m.TagPtr == 0 }

// aff70 is 1aff70: it divides the morphs of a sentence into prosodic words,
// computing the accent of each as it is closed. It reports false when the
// sentence has nothing to build.
func (d *Document) aff70(s *Sentence) bool {
	if s.Chunk == nil || s.Chunk.First == nil {
		return false
	}
	endM := s.endMorph()
	curPW := d.insertPWord(endM)
	if curPW == nil || s.Chunk == nil {
		return false
	}
	m := s.Chunk.First
	for ; m != nil; m = m.Next {
		if m == endM || m.POS&0xffff0000 == 0xb080000 {
			break
		}
		if (m.Moras > 0 || m.TagPtr != 0) && (m.TagPtr == 0 || m.TagLen != 0) {
			break
		}
	}
	if m == nil {
		return false
	}
	curPW.First = m
	m.PWord = curPW
	if m == endM {
		return false
	}
	for !(m == endM || m.Moras > 0 || m.TagPtr != 0) {
		m = m.Next
		if m == nil {
			return false
		}
	}
	var prevPW *PWord
	var marker int32 // carried S2/S3 marker (local_168 / uVar21)
	m = m.Next
	for {
		if m == nil {
			return true
		}
		val := int32(0)
		prev := m.Prev
		decided := false // true when the value was set and processing continues at 1b048f
		switch {
		case m == endM:
			val = 1
			if marker < 0 {
				val = marker
			}
			decided = true
		case prev != nil && prev.Kind == 0x15:
			val, decided = 1, true
		case prev != nil && prev.Kind == 0x11 && !silentTagFree(m):
			val, decided = 1, true
		case m.POS&0xffff0000 == 0xb080000:
			val, decided = 2, true
		case prev != nil && prev.POS&0xffff0000 == 0xb080000:
			val, decided = 1, true
		case prev != nil && prev.Kind == 0x10 && !silentTagFree(m):
			val, decided = -2, true
		default:
			val, decided = d.decideLocal(m, prev)
		}
		if decided && val != 0 && m != endM && m.POS&0xffff0000 != 0xb080000 {
			if cancelSplit(d, prev, m, curPW, endM) {
				val = 0
			}
		}
		if val == 0 {
			d.propagateKind(prev, m)
			m = m.Next
			continue
		}
		// start a new prosodic word at m
		if m != endM {
			d.insertPWord(m)
		}
		old := marker
		newMarker := int32(0)
		limitA := d.LimitA
		for {
			uVar9 := curPW
			var newP *PWord
			if 2*float64(limitA) < float64(uVar9.Moras) {
				newP = d.splitLong(uVar9, limitA)
			}
			d.ac9b0(prevPW, uVar9)
			if prevPW != nil {
				switch old {
				case -1:
					if prevPW.Down < 1 || uVar9.Down == 0 || uVar9.Down == uVar9.Moras {
						curPW = d.mergeBack(s, prevPW, uVar9)
					}
				case -2:
					if !ac870(prevPW, uVar9) {
						curPW = d.mergeBack(s, prevPW, uVar9)
					} else if prevPW.Next != nil && prevPW.Next.First != nil && prevPW.Next.First.Prev != nil {
						prevPW.Next.First.Prev.Kind = 0x10
					}
				}
			}
			if val < 0 {
				newMarker = val
			}
			if newP == nil {
				break
			}
			prevPW = newP.Prev
			curPW = newP
		}
		if m == endM {
			return true
		}
		next := curPW.Next
		if next == s.endPWord() {
			return true
		}
		prevPW, marker, curPW = curPW, newMarker, next
		m = m.Next
	}
}

// splitLong is the splitting of an overlong prosodic word inside 1aff70: it
// finds a boundary after about LimitA moras and starts a new word there.
func (d *Document) splitLong(p *PWord, limit int32) *PWord {
	acc := int32(0)
	end := p.endMorph()
	q := p.First
	for q != nil && q != end {
		acc += q.Moras
		if limit < acc {
			break
		}
		if limit-5 < acc && q.POS == 0x1040000 && q.Prev != nil && q.Prev.POS != 0x1040000 {
			break
		}
		q = q.Next
		if q == nil {
			break
		}
	}
	if 6 < p.Moras-acc {
		var nextFirst *Morph
		if p.Next != nil {
			nextFirst = p.Next.First
		}
		if q != nextFirst {
			after := q.Next
			if after != nextFirst {
				q.Kind = 0x13
				if after != nil {
					return d.insertPWord(after)
				}
			}
		}
	}
	return nil
}

// mergeBack removes cur and recomputes prev (label 1b09eb). It returns the
// prosodic word that is current afterwards.
func (d *Document) mergeBack(s *Sentence, prev, cur *PWord) *PWord {
	if first := cur.First; first == nil || len(first.Origins) == 0 {
		d.removePWord(cur)
		if prev == s.PWord {
			d.ac9b0(nil, prev)
		} else {
			d.ac9b0(prev.Prev, prev)
		}
		return prev
	}
	if prev.Next != nil && prev.Next.First != nil && prev.Next.First.Prev != nil {
		if last := prev.Next.First.Prev; last.Kind == 0x10 {
			last.Kind = 0x11
		}
	}
	return cur
}

// decideLocal covers the part of 1aff70 that consults the previous morph's
// kind and 1b0ba0 for an ordinary morph. It returns the decision and whether
// it was decided (false means "join", handled at 1b06b8).
func (d *Document) decideLocal(m, prev *Morph) (int32, bool) {
	if prev != nil && prev.Kind == 0xc && !silentTagFree(m) {
		return 0, false
	}
	if prev == nil {
		if m.Kind-0x18 < 2 {
			return 0, false
		}
		return b0ba0(nil, m, d.Flag21), true
	}
	if prev.Kind == 0xf {
		return 1, true
	}
	if uint32(prev.Kind)-0x18 > 1 {
		if uint32(m.Kind)-0x18 < 2 {
			return d.tagDecision(prev, m)
		}
		if prev.Kind == 0x1e || uint32(prev.Kind)-0x1c < 2 {
			return 1, true
		}
		v := b0ba0(prev, m, d.Flag21)
		return v, v != 0
	}
	return d.tagDecision(prev, m)
}

// tagDecision is the 1b03ef branch for tag boundary kinds 0x18 and 0x19.
func (d *Document) tagDecision(prev, m *Morph) (int32, bool) {
	cur, pk := m.Kind, prev.Kind
	prevSilent := silentTagFree(prev)
	curSilent := silentTagFree(m)
	var v int32
	switch {
	case (pk == 0x19 || prevSilent) && cur != 0x19:
		v = 1
	case pk == 0x18:
		if (cur != 0x19 || m.TagLen != 0) && !curSilent {
			v = 1
		}
	default:
		if cur != 0x19 {
			v = 1
		} else if m.TagLen != 0 {
			v = 1
		}
	}
	return v, v != 0
}

// cancelSplit reports whether the split decided at m is cancelled by the
// checks at 1b04a0..1b06a2 (silent neighbours or nothing audible before/after).
func cancelSplit(d *Document, prev, m *Morph, curPW *PWord, endM *Morph) bool {
	k := func(x *Morph) uint32 { return uint32(x.Kind) }
	bVar23 := true
	if prev == nil ||
		(silentTagFree(prev) && k(prev)-0x18 > 1 || k(prev)-0x1c < 2) ||
		(prev.Kind == 0x19 && prev.TagLen == 0) ||
		((m.Moras > 0 || m.TagPtr != 0 || k(m)-0x18 < 2) && (m.Kind != 0x19 || m.TagLen != 0)) {
		bVar23 = false
	}
	bVar22 := true
	if prev == nil ||
		((prev.Moras > 0 || prev.TagPtr != 0 || k(prev)-0x18 < 2) && k(prev)-0x1c > 1 && (prev.Kind != 0x19 || prev.TagLen != 0)) ||
		((m.Moras > 0 || m.TagPtr != 0 || k(m)-0x18 < 2) && (m.Kind != 0x19 || m.TagLen != 0)) {
		bVar22 = false
	}
	cVar5 := true
	if curPW.First != m {
		cVar5 = afec0(curPW.First, m)
	}
	bVar7 := true
	for x := m; x != endM; x = x.Next {
		if x == nil {
			break
		}
		if (x.Moras > 0 || x.TagPtr != 0 || k(x)-0x18 < 2) && (x.Kind != 0x19 || x.TagLen != 0) {
			bVar7 = false
			break
		}
	}
	return bVar23 || bVar22 || cVar5 || bVar7
}

// propagateKind is the 1b06b8 block: a silent morph takes over the kind of the
// stronger preceding boundary.
func (d *Document) propagateKind(prev, m *Morph) {
	if prev == nil {
		return
	}
	k := func(x *Morph) uint32 { return uint32(x.Kind) }
	if k(prev)-0x18 < 2 || m.Moras > 0 || m.TagPtr != 0 || k(m)-0x18 < 2 {
		if prev.Kind != 0x19 || prev.TagLen != 0 || m.Moras > 0 || m.TagPtr != 0 {
			return
		}
		if m.Kind == 0x18 || m.Kind == 0x19 {
			return
		}
		afc70(prev, m)
		return
	}
	if !d.Flag21 || rankOf(prev.Kind) <= rankOf(m.Kind) {
		if kindStrength(prev.Kind) <= kindStrength(m.Kind) {
			return
		}
	}
	afc70(prev, m)
}
