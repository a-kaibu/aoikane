package numeral

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// readMixedZeroUnits mirrors the unit portion of 100a90. Digit-script
// transitions and multi-digit -> unit transitions carry ｂ; unit -> digit
// transitions concatenate the sounds before table lookup. A large unit stays
// attached to the preceding numeric table word within its fragment.
func (r *Rules) readMixedZeroUnits(req Request) ([]Reading, error) {
	type segment struct {
		text  string
		pause bool
	}
	var segments []segment
	// Multi-digit coefficients before an interior small unit are separate
	// 10b010 fragments. A trailing Arabic coefficient/unit is expanded by
	// expandedUnitInteger before this generic path (２１百 -> ２１００).
	pairedDigits := req.reiTail || strings.ContainsRune(req.Number, '零')
	buffer, kind, pause := "", "", false
	flush := func(nextPause bool) {
		if buffer != "" {
			segments = append(segments, segment{buffer, pause})
		}
		buffer, pause = "", nextPause
	}
	for i, c := range req.Number {
		next := r.Digits[string(c)].Type
		if _, ok := smallUnits[c]; ok || strings.ContainsRune(string(bigUnits), c) {
			next = "Unit"
		}
		if next != "Jap" && next != "Eng" && next != "Unit" {
			return nil, fmt.Errorf("unsupported small-unit numeral %q", req.Number)
		}
		switch {
		case (kind == "Jap" || kind == "Eng") && next == "Unit":
			_, small := smallUnits[c]
			if strings.ContainsRune(buffer, '零') || small && utf8.RuneCountInString(buffer) > 1 &&
				(kind == "Jap" || pairedDigits || strings.Trim(buffer, "０〇") == "" || !leadingZeroDigitRun(buffer) || req.connectorStream || i+utf8.RuneLen(c) < len(req.Number)) {
				flush(true)
			}
		case (kind == "Jap" || kind == "Eng") && (next == "Jap" || next == "Eng") && next != kind:
			flush(true)
		case kind == "Unit" && (next == "Jap" || next == "Eng"):
			flush(false)
		}
		buffer += string(c)
		kind = next
	}
	flush(false)
	type block struct {
		parts []string
		pause bool
	}
	var blocks []block
	for _, s := range segments {
		if len(blocks) == 0 || s.pause || leadingZeroDigitRun(s.text) {
			blocks = append(blocks, block{pause: s.pause})
		}
		blocks[len(blocks)-1].parts = append(blocks[len(blocks)-1].parts, s.text)
	}
	var prefix []Group
	for bi, b := range blocks {
		block := b.parts
		partReq := req
		if bi < len(blocks)-1 {
			partReq.Counter = ""
		}
		var readings []Reading
		var err error
		if len(block) == 1 {
			partReq.Number = block[0]
			if bi > 0 && writtenLargeUnits(block[0]) && strings.ContainsRune(strings.Join(blocks[bi-1].parts, ""), '零') {
				readings, err = r.connectorUnitWords(partReq)
			} else if u, ok := r.Units[block[0]]; ok && partReq.Counter != "" {
				c := chunk{Read: u.Read, Sound: u.Sound, Key: block[0], TkKey: block[0], numericFragment: true}
				readings, err = r.withCounter(partReq, []chunk{c}, false)
			} else if bi+1 < len(blocks) && blocks[bi+1].pause && !pairedDigits && fullWidthDigits(block[0]) && !leadingZeroDigitRun(block[0]) {
				// The final table word keeps its fall before the boundary.
				// Earlier words still apply their continuation flag: ニジュー
				// stays accented in ２１ｂ百, but ハチジュー flattens in ８２ｂ百.
				var sound string
				sound, err = r.integerFragmentSound(block[0])
				if err == nil {
					var cs []chunk
					cs, err = r.numericSoundChunks(sound)
					var gs []Group
					for i, c := range cs {
						g := r.accentOf(c)
						if i+1 < len(cs) {
							r.flatBeforeMore(c, &g)
						}
						gs = append(gs, g)
					}
					readings = []Reading{{Groups: gs}}
				}
			} else {
				readings, err = r.read(partReq, true)
			}
		} else if pairedZeroLargeUnit(block[0]) {
			// A leading-zero digit pair stays attached to its large unit
			// before later digits (０９万２), rather than being tokenized
			// again as a separate ゼロ and キューマン.
			var head []Reading
			head, err = r.read(Request{Number: block[0], NoSpecial: true}, true)
			if err == nil {
				partReq.Number = strings.Join(block[1:], "")
				partReq.NoSpecial = true
				readings, err = r.read(partReq, false)
				for i := range readings {
					readings[i].Groups = append(append([]Group(nil), head[0].Groups...), readings[i].Groups...)
				}
			}
		} else if last := block[len(block)-1]; strings.ContainsRune(last, '零') && utf8.RuneCountInString(last) > 1 && r.Units[string([]rune(block[0])[0])].Type == "Unit" {
			// 1043f0's paired literal-zero tail keeps its own group;
			// retokenizing チョーレーゼロ would lose レーゼロ's accent.
			partReq.Number = strings.Join(block, "")
			readings, err = r.unitZeroTail(partReq)
		} else {
			var sound string
			fragmentEnds := map[int]bool{}
			for pi, part := range block {
				plain, e := r.read(Request{Number: part}, true)
				if e != nil {
					return nil, e
				}
				for _, g := range plain[0].Groups {
					if g.Pause {
						return nil, fmt.Errorf("unexpected internal pause in numeric unit block %q", part)
					}
					sound += g.Sound
				}
				if pi+1 < len(block) && utf8.RuneCountInString(block[pi+1]) > 1 {
					fragmentEnds[len(sound)] = true
				}
			}
			var chunks []chunk
			chunks, err = r.numericSoundChunks(sound)
			if err == nil {
				// Preserve a word ending before a multi-digit 10b010 fragment.
				// Internal value chunks still flatten: ８十/１１ keeps
				// ハチジュー's fall, but 百/６７ flattens ロクジュー.
				end := 0
				for i := range chunks {
					_, word := r.chunkWord(chunks[i])
					end += len(word)
					chunks[i].fragmentBoundary = fragmentEnds[end]
				}
				if partReq.Counter != "" {
					readings, err = r.withCounter(partReq, chunks, false)
				} else {
					var groups []Group
					for i, c := range chunks {
						g := r.accentOf(c)
						if i < len(chunks)-1 {
							r.flatBeforeMore(c, &g)
						}
						groups = append(groups, g)
					}
					readings = []Reading{{Groups: groups}}
				}
			}
		}
		if err != nil {
			return nil, err
		}
		if bi == len(blocks)-1 {
			for i := range readings {
				readings[i].Groups = append(append([]Group(nil), prefix...), readings[i].Groups...)
			}
			return readings, nil
		}
		prefix = append(prefix, readings[0].Groups...)
		if blocks[bi+1].pause {
			prefix = append(prefix, Group{Read: "ｂ", Pause: true})
		}
	}
	return nil, fmt.Errorf("empty small-unit numeral")
}

// numericSoundChunks performs ordered table-prefix lookup on a concatenated
// small-unit sound. For example, ジュー + ロクジュー + ナナ becomes the two
// table words ジューロク / ジューナナ, retaining 16/17's counter keys and Parts.
func (r *Rules) numericSoundChunks(sound string) ([]chunk, error) {
	templates := map[string]chunk{}
	zero := r.digitWord(0)
	templates[zero.Sound] = chunk{Read: zero.Read, Sound: zero.Sound, Key: "〇", TkKey: "〇"}
	if rei, ok := r.Digits["零"]; ok {
		templates[rei.Sound] = chunk{Read: rei.Read, Sound: rei.Sound, Key: "零", TkKey: "零"}
	}
	// An explicit 一 before 千 survives 10b010 and assimilates to イッセン;
	// the positional value 1000 instead has the bare セン template.
	one, thousand := r.digitWord(1), r.Units["千"]
	explicitThousand := chunk{Read: geminate(one.Read) + thousand.Read, Sound: geminate(one.Sound) + thousand.Sound,
		Key: "千", TkKey: "一", numericFragment: true}
	templates[explicitThousand.Sound] = explicitThousand
	for d := 1; d <= 9; d++ {
		for _, value := range []int{d, d * 10, d * 100, d * 1000, d + 10} {
			for _, c := range r.groupChunks(value) {
				c.numericFragment = true
				templates[c.Sound] = c
			}
		}
	}
	var bases []chunk
	for _, c := range templates {
		bases = append(bases, c)
	}
	for _, unit := range bigUnits {
		key := string(unit)
		u := r.Units[key]
		for _, base := range bases {
			c := attachLargeUnit(base, key)
			templates[c.Sound+u.Sound] = c
			if c.Sound != base.Sound {
				stem := c
				stem.Key, stem.TkKey, stem.Unit = base.Key, base.TkKey, ""
				templates[stem.Sound] = stem
			}
		}
		templates[u.Sound] = chunk{Read: u.Read, Sound: u.Sound, Key: key, TkKey: key, numericFragment: true}
	}
	var chunks []chunk
	for sound != "" {
		matched := false
		for _, w := range r.Accents {
			if w.Sound == "" || !strings.HasPrefix(sound, w.Sound) {
				continue
			}
			c, ok := templates[w.Sound]
			if !ok {
				return nil, fmt.Errorf("numeric table word %q has no small-unit key", w.Sound)
			}
			// A full table word such as ニチョー is a stem in its own
			// right. It can take one following large unit (ニチョーマン).
			if c.Unit != "" {
				u := r.Units[c.Unit]
				c.Read, c.Sound = c.Read+u.Read, c.Sound+u.Sound
				c.Unit = ""
				c.tableStem = true
			}
			chunks = append(chunks, c)
			sound = strings.TrimPrefix(sound, w.Sound)
			if c.Unit == "" {
				for _, unit := range bigUnits {
					key := string(unit)
					u := r.Units[key]
					if u.Sound != "" && strings.HasPrefix(sound, u.Sound) {
						last := &chunks[len(chunks)-1]
						*last = attachLargeUnit(*last, key)
						sound = strings.TrimPrefix(sound, u.Sound)
						break
					}
				}
			}
			matched = true
			break
		}
		if !matched {
			return nil, fmt.Errorf("unknown numeric sound %q", sound)
		}
	}
	for i := 0; i+1 < len(chunks); i++ {
		next := chunks[i+1]
		if next.Unit == "" && next.Key == next.TkKey && r.Units[next.Key].Type == "Unit" {
			chunks[i].literalUnitBoundary = true
		}
	}
	return chunks, nil
}

// A leading-zero stream after a small unit retains its paired accent groups
// even though the reading has no written pause (十０３ is ジューゼロサン).
func unitBeforeZero(number string) bool {
	var previous rune
	for _, c := range number {
		if _, unit := smallUnits[previous]; (unit || strings.ContainsRune(string(bigUnits), previous)) && (c == '０' || c == '〇') {
			return true
		}
		previous = c
	}
	return false
}
