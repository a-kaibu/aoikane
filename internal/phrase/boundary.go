package phrase

// b0ba0 is 1b0ba0: the decision between the previous morph p1 and the current
// morph p2. A non-zero result starts a new prosodic word (1; -1 and -2 mark the
// "S2" and "S3" connection rules); zero keeps both words together. noRules is
// the frontend flag at +0x21: when set only the explicit checks apply.
// The native function also records a label in the annotation of p1 which does
// not influence the result and is not reproduced.
func b0ba0(p1, p2 *Morph, noRules bool) int32 {
	if p1 == nil || p2 == nil {
		return 1
	}
	next := p2.Next // local_98
	if p2.silent() && (p2.POS+0xf4f90000)&0xfffeffff != 0 {
		return 0
	}
	if p1.silent() && (p1.POS+0xf4f90000)&0xfffeffff != 0 {
		return 1
	}
	lead := ""
	for _, r := range p2.Pronunciation() {
		lead = string(r)
		break
	}
	if (lead == "ー" || lead == "ッ") && !p1.silent() {
		return 0
	}
	if p1.Kind != 0 && kindStrength(p1.Kind) > 0 && kindClass(p1.Kind) == 2 {
		return 1
	}
	if c := p2.SelectedCandidate(); c != nil {
		switch rule, _, _ := ParseRule(c.Accent, c.Connection, 0, -1); rule {
		case RuleS1:
			return 1
		case RuleS2:
			return -1
		case RuleS3:
			return -2
		}
	}
	if p1.POS&0xff000000 == 0x2000000 {
		if c := p1.SelectedCandidate(); c != nil {
			if rule, _, _ := ParseRule(c.Accent, c.Connection, 0, -1); rule == RuleP3 {
				return 1
			}
		}
	}
	if noRules {
		return 0
	}
	class1 := p1.POS & 0xff000000
	class2 := p2.POS & 0xff000000
	switch class1 {
	case 0x2000000:
		return 0 // prefix other than P3
	case 0x3000000:
		if p1.POS2&0xff00 == 0x600 && (p2.POS == 0x3010000 || p2.POS == 0x4010000) {
			return 0 // compound verb/adjective, rules 7 and 8
		}
		if p1.p16b750() && class2 == 0x3000000 {
			if p1.Moras >= 4 || p1.Base == "する" {
				if p2.Moras > 3 {
					return 1 // compound verb, rule 9
				}
			}
			return 0
		}
	}
	// nouns and the remaining classes
	if !p1.p16b950() || (p1.POS+0xfefc0000)&0xfffdffff == 0 || class2 != 0x3000000 || p2.POS2&0xff00 != 0x600 ||
		next == nil || next.POS != 0xb020000 {
		if class1 == 0x2000000 && p2.p16b950() || p1.p16b950() && p2.p16b950() || p1.p16b950() && p2.p16b980() {
			return compoundNoun(p1, p2, next)
		}
		if !p2.p16b7f0() && class2 != 0x2000000 {
			return particleRules(p1, p2, next)
		}
		// independent word or prefix, rules 23 and 24
		if p2.Base != "～" || p2.Moras < 1 {
			return 1
		}
		return 0
	}
	return 0 // compound verb, rule 10
}

// compoundNoun covers rules 11 to 21.
func compoundNoun(p1, p2, next *Morph) int32 {
	if p2.POS == 0x1060000 {
		return 1
	}
	if p1.POS == 0x1060000 && !p2.p16b980() && !p2.p16b7b0() {
		if p2.Base != "生まれ" && p2.Base != "前" {
			return 1
		}
	}
	if ((p2.POS&0xffff0000)+0xfeff0000)&0xfffeffff == 0 {
		return 1
	}
	if p1.POS&0xffffff00 == 0x1010100 && !p2.p16b980() {
		return 1
	}
	if p1.POS&0xffff0000 == 0x1020000 && !p2.p16b980() {
		return 1
	}
	if (p1.POS == 0x1030000 || p1.POS == 0x1070100 || p1.POS == 0x1090600) && !p2.p16b980() {
		if p2.Base != "生まれ" {
			return 1
		}
	}
	if p1.POS&0xffff0000 == 0x1070000 && !p2.p16b980() {
		return 1
	}
	if p1.POS == 0x10a0000 || p2.POS == 0x10a0000 {
		return 1
	}
	if !p1.p16b950() && p2.POS == 0x1040000 && next != nil && next.POS3 == 0x200 {
		return 1
	}
	return 0
}

// particleRules covers rules 25 to 30.
func particleRules(p1, p2, next *Morph) int32 {
	class1 := p1.POS & 0xff000000
	class2 := p2.POS & 0xff000000
	switch class1 {
	case 0x8000000:
		if class2 == 0x9000000 {
			if p2.POS3&0xffffff00 < 0xa01 || p2.POS3 == 0xc00 {
				return 1
			}
			return 0
		}
		if class2 != 0x8000000 {
			if p1.Base != "て" && p1.Base != "で" {
				return 1
			}
			if p2.p16b7b0() {
				switch {
				case p2.Base == "くる", p2.Base == "ある", p2.Base == "おく",
					p2.Base == "くれる" && p2.POS2 == 0x500, p2.Base == "ちょうだい":
				default:
					return 1
				}
				if next != nil && next.Base != "ます" && next.Base != "ない" {
					return 0
				}
			}
			return 1
		}
	case 0x9000000:
		if class2 == 0x9000000 {
			if p1.Base == "だ" && p2.Base == "ない" {
				return 1
			}
			return 0
		}
		if !p2.p16b780() {
			if p2.POS == 0x1070000 {
				if p2.Moras == 1 {
					return 0
				}
			} else if p2.POS == 0x1070200 {
				return 0
			}
			return 1
		}
	}
	return 0
}
