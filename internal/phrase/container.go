package phrase

import "errors"

// The prosodic-word list is an intrusive list held by the document (native
// container at +0x1f0). The operations below mirror 159e10, 190d60, 190b90,
// 1908d0 and 18fff0 including their quirks.

// endMorph is 1280e0: the first morph of the next sentence.
func (s *Sentence) endMorph() *Morph {
	if s.Next == nil || s.Next.Chunk == nil {
		return nil
	}
	return s.Next.Chunk.First
}

// endPWord is 173660: the first prosodic word of the next sentence.
func (s *Sentence) endPWord() *PWord {
	if s.Next == nil {
		return nil
	}
	return s.Next.PWord
}

// find159e10 returns the prosodic word that follows the one containing key,
// walking back from the tail; nil when key is not found.
func (d *Document) find159e10(key *Morph) *PWord {
	cur := d.PWTail
	if cur == nil || key == nil {
		return nil
	}
	var boundary *Morph
	if cur.Prev != nil {
		boundary = cur.Prev.First
	}
	if cur.First == nil {
		return nil
	}
	for m := cur.First.Prev; m != nil; m = m.Prev {
		if m == key {
			return cur
		}
		if boundary == m {
			cur = cur.Prev
			if cur == nil || cur.Prev == nil {
				boundary = nil
			} else {
				boundary = cur.Prev.First
			}
		}
	}
	return nil
}

// appendBeforeTail is 190d60.
func (d *Document) appendBeforeTail(n *PWord) *PWord {
	if d.PWTail != nil {
		prev := d.PWTail.Prev
		n.Prev, n.Next = prev, d.PWTail
		d.PWTail.Prev = n
		d.PWCount++
		if prev != nil {
			prev.Next = n
			return n
		}
		d.PWHead = n
		return n
	}
	n.Prev, n.Next = nil, nil
	d.PWCount++
	d.PWHead = nil
	d.PWTail = n
	return n
}

// insertPWord is 1908d0: it creates a prosodic word that starts at key.
func (d *Document) insertPWord(key *Morph) *PWord {
	n := &PWord{First: key, Down: 0, Up: 0, Moras: -1}
	found := d.find159e10(key)
	if found == nil {
		if key != d.EOS || d.PWTail == nil || d.End == nil {
			return nil
		}
		if p := d.PWTail.Prev; p != nil && p.First == d.EOS {
			return nil
		}
		last := d.End.Prev
		n = d.appendBeforeTail(n)
		n.First = d.EOS
		if last != nil && last.PWord == d.PWTail {
			last.PWord = n
			n.Owner = last
			return n
		}
		n.Owner = nil
		return n
	}
	if key.PWord != nil {
		return nil
	}
	appended := false
	switch {
	case found == d.PWTail && d.PWHead == nil:
		n = d.appendBeforeTail(n)
		appended = true
	case found == d.PWTail:
	default:
		head := d.PWHead
		if d.PWCount == 1 {
			if head != found || head == nil {
				return nil
			}
		} else {
			for {
				if head == nil || head == d.PWTail {
					return nil
				}
				if head == found {
					break
				}
				head = head.Next
			}
		}
	}
	if !appended {
		prev := found.Prev
		n.Prev = prev
		if prev == nil {
			if d.PWHead == found {
				d.PWHead = n
			}
		} else {
			prev.Next = n
		}
		n.Next = found
		found.Prev = n
		d.PWCount++
	}
	n.Owner = nil
	n.First.PWord = n
	return n
}

var errPWordNotFound = errors.New("prosodic word is not in the list")

// removePWord is 18fff0. It returns the prosodic word that followed.
func (d *Document) removePWord(p *PWord) *PWord {
	inList := false
	if d.PWCount == 1 {
		inList = d.PWHead == p && p != nil
	} else {
		for q := d.PWHead; q != nil && q != d.PWTail; q = q.Next {
			if q == p {
				inList = true
				break
			}
		}
	}
	if !inList {
		return nil
	}
	if c := p.First; c != nil && c.PWord == p {
		c.PWord = nil
	}
	if o := p.Owner; o != nil && o.PWord == p {
		if p.Next.Owner == nil {
			o.PWord = p.Next
			p.Next.Owner = o
		} else {
			o.PWord = nil
		}
	}
	next, prev := p.Next, p.Prev
	if prev == nil {
		if p == d.PWHead {
			d.PWHead = next
		}
	} else {
		prev.Next = next
	}
	d.PWCount--
	if next != nil {
		next.Prev = prev
	}
	return next
}
