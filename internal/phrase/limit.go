package phrase

import "math"

// kindOf is 16ce60: the first non-zero boundary kind found walking back from
// the last morph of the prosodic word, stopping at the first audible morph.
func (p *PWord) kindOf() int32 {
	if p.Next == nil || p.Next.First == nil || p.First == nil {
		return 0
	}
	stop := p.First.Prev
	for m := p.Next.First.Prev; m != nil && m != stop; m = m.Prev {
		if m.Kind != 0 {
			return m.Kind
		}
		if m.Moras > 0 || m.TagPtr != 0 {
			return 0
		}
	}
	return 0
}

// kind2Of is 16d300: the sentence-end style recorded on the last audible
// morph of the final prosodic word of a sentence.
func (p *PWord) kind2Of() int32 {
	if p.Next == nil || p.Next.First == nil || p.First == nil {
		return 0
	}
	last := p.Next.Next == nil || p.Next.Owner != nil
	if !last {
		return 0
	}
	stop := p.First.Prev
	for m := p.Next.First.Prev; m != nil && m != stop; m = m.Prev {
		if k := m.Kind2; k >= 0x1f && k <= 0x28 {
			return k
		}
		if m.Moras > 0 || m.TagPtr != 0 {
			return 0
		}
	}
	return 0
}

// ahead is 167d40: the prosodic word n steps ahead, nil at the end.
func (p *PWord) ahead(n int) *PWord {
	l := p.Next
	for i := 1; l != nil && i < n; i++ {
		l = l.Next
	}
	if l != nil && l.Next == nil {
		return nil
	}
	return l
}

func absStrengthSet(s float32) bool { return 1e-10 <= math.Abs(float64(s)) }

// af570 is 1af570: boundary strengths between prosodic words and, for the
// Kansai configuration, the limits on phrase length.
func (d *Document) af570(s *Sentence) {
	end := s.endPWord()
	count := 0
	for p := s.PWord; p != nil && p != end; p = p.Next {
		count++
	}
	if count < 2 {
		return
	}
	lastMorph := (*Morph)(nil)
	if em := s.endMorph(); em != nil {
		lastMorph = em.Prev
	}
	for p := s.PWord; p != nil && p != end; p = p.Next {
		if p.kindOf() == 0x19 {
			continue
		}
		if p.kindOf() == 0x18 {
			if p.Next != nil && p.Next.First != nil && p.Next.First.Prev != nil && p.Next.First.Prev.Kind != 0 {
				continue
			}
		}
		if p.kindOf() == 0x1e {
			continue
		}
		var m *Morph
		if p.Next != nil && p.Next.First != nil {
			m = p.Next.First.Prev
		}
		stop := (*Morph)(nil)
		if p.First != nil {
			stop = p.First.Prev
		}
		for ; ; m = m.Prev {
			if m == nil || m == stop || m.Moras > 0 || m.TagPtr != 0 || m.POS&0xffff0000 == 0xb080000 {
				break
			}
			if m.POS == 0xb020000 && m != lastMorph {
				p.Flag39 = true
				break
			}
		}
	}
	for p := s.PWord; p != nil && p != end; p = p.Next {
		k := p.kindOf()
		if k == 0x1e {
			continue
		}
		if p.Flag39 && kindClass(0x16) == 2 {
			p.Strength = kindStrength(0x16)
			p.Flag38 = kindStrength(0x16) > 0
			if uint32(k)-2 > 2 && k != 9 {
				if p.Next != nil && p.Next.First != nil && p.Next.First.Prev != nil {
					p.Next.First.Prev.Kind = 0x16
				}
			}
		}
		if st := kindStrength(k); p.Strength < st {
			p.Strength = st
			p.Flag38 = kindClass(k) == 2
		}
	}
	if d.Flag21 {
		return
	}
	d.limitA(s, end)
	d.limitB(s, end)
}

// markSplit closes the prosodic word cand as a strong boundary.
func markSplit(cand *PWord, strength float32) {
	cand.Flag38 = true
	cand.Strength = strength
	if cand.Next != nil && cand.Next.First != nil && cand.Next.First.Prev != nil {
		cand.Next.First.Prev.Kind = 7
	}
}

// lastMorphOf returns the last morph of the prosodic word.
func (p *PWord) lastMorph() *Morph {
	if p.Next == nil || p.Next.First == nil {
		return nil
	}
	return p.Next.First.Prev
}

func (d *Document) limitA(s *Sentence, end *PWord) {
	var cand *PWord
	sum := int32(0)
	for p := s.PWord; p != nil && p != end; p = p.Next {
		cur := sum + p.Moras
		if p.Moras < 1 {
			cur = sum
		}
		if !p.Flag38 {
			k := p.kindOf()
			if k != 0x1e {
				ahead := p.ahead(2)
				if p.Strength <= 0 && k == 0x13 {
					next := p.Next
					skip := false
					if last := p.lastMorph(); last != nil {
						if last.Base == "を" && last.POS == 0x8000000 {
							skip = true
						}
						if !skip && last.Base == "の" && last.POS == 0x8000000 {
							skip = true
						}
					}
					if !skip {
						limit := d.LimitB
						if limit < cur {
							take := false
							if ahead == nil {
								take = !(next.Moras <= limit)
							} else if ahead == end || next.Moras+ahead.Moras <= limit || absStrengthSet(next.Strength) {
								if ahead == end {
									take = !(next.Moras <= limit)
								}
							} else {
								take = true
							}
							if take {
								cand = p
							}
						}
					}
				}
				if d.LimitA < cur && cand != nil {
					markSplit(cand, 0.3)
					p = cand
					cand = nil
				}
				if p.Flag38 {
					cur = 0
					cand = nil
				}
			}
		} else {
			cur = 0
			cand = nil
		}
		sum = cur
	}
}

func (d *Document) limitB(s *Sentence, end *PWord) {
	var cand *PWord
	sum := int32(0)
	for p := s.PWord; p != nil && p != end; p = p.Next {
		cur := sum + p.Moras
		if p.Moras < 1 {
			cur = sum
		}
		if !p.Flag38 {
			k := p.kindOf()
			if k != 0x1e {
				var nextFirst, last *Morph
				if p.Next != nil {
					nextFirst = p.Next.First
					if nextFirst != nil {
						last = nextFirst.Prev
					}
				}
				ahead := p.ahead(2)
				if p.Strength <= 0 && kindClass(0xb) != 0 && k == 1 && last != nil {
					ok := false
					switch pos := last.POS; {
					case pos == 0x3010000:
						ok = last.POS2 == 0x100
					case pos == 0x3000000 && last.POS2 == 0x100:
						ok = true
					case pos == 0x9000000:
						ok = last.POS2 == 0x100
					}
					if ok {
						limit := d.LimitB * 2
						if limit < cur {
							next := p.Next
							pass := false
							if ahead == nil {
								pass = !(next.Moras <= limit)
							} else if ahead == end || next.Moras+ahead.Moras <= limit || absStrengthSet(next.Strength) {
								if ahead == end {
									pass = !(next.Moras <= limit)
								}
							} else {
								pass = true
							}
							if pass && nextFirst != nil {
								second := nextFirst.Next
								if nextFirst.p16b950() && nextFirst.Surface != "よう" && second != nil &&
									second.POS != 0x8000000 && second.POS != 0x1070000 {
									cand = p
								}
							}
						}
					}
				}
				if d.LimitA < cur && cand != nil {
					markSplit(cand, 0.22)
					p = cand
					cand = nil
				}
				if p.Flag38 {
					cur = 0
					cand = nil
				}
			}
		} else {
			cur = 0
			cand = nil
		}
		sum = cur
	}
}
