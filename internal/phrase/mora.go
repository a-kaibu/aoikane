package phrase

// The native engine treats pronunciations as strings of two-byte CP932
// characters. Here they are UTF-8, so the helpers work on runes; every
// pronunciation character corresponds to exactly one native two-byte unit.

// smallKana are the characters that do not start a new mora (1eb540):
// ァィゥェォャュョヮヵヶ゜
var smallKana = map[rune]bool{
	'ァ': true, 'ィ': true, 'ゥ': true, 'ェ': true, 'ォ': true,
	'ャ': true, 'ュ': true, 'ョ': true, 'ヮ': true, 'ヵ': true, 'ヶ': true, '゜': true,
}

// startsMora is 1eb540: true unless the character is a small kana.
func startsMora(r rune) bool { return !smallKana[r] }

// isSpecialMora is 1eb890: ン, ッ or ー.
func isSpecialMora(r rune) bool { return r == 'ン' || r == 'ッ' || r == 'ー' }

// isVowelKana is 1eb440: ア, イ, ウ, エ or オ.
func isVowelKana(r rune) bool {
	return r == 'ア' || r == 'イ' || r == 'ウ' || r == 'エ' || r == 'オ'
}

// firstSpecial reports isSpecialMora for the first character of s.
func firstSpecial(s string) bool {
	for _, r := range s {
		return isSpecialMora(r)
	}
	return false
}

// lastMoras is 16af40: the pronunciation suffix that holds the last n moras
// (counted by characters that start a mora), or "" when there are fewer.
func lastMoras(m *Morph, n int) string {
	rs := []rune(m.Pronunciation())
	count := 0
	for i := len(rs) - 1; i >= 0; i-- {
		if startsMora(rs[i]) {
			count++
		}
		if count >= n {
			return string(rs[i:])
		}
	}
	return ""
}

// moraRange is 16ac60: the characters of moras [start, start+count) of the
// morph's pronunciation. A count of zero or less means "to the end".
func moraRange(m *Morph, start, count int) string {
	rs := []rune(m.Pronunciation())
	if start < 0 || len(rs) == 0 {
		return ""
	}
	seen := 0
	begin := -1
	for i, r := range rs {
		if startsMora(r) {
			seen++
		}
		if seen > start {
			begin = i
			break
		}
	}
	if begin < 0 {
		return ""
	}
	end := len(rs)
	if count > 0 {
		for i := begin + 1; i < len(rs); i++ {
			if startsMora(rs[i]) {
				seen++
			}
			if seen > start+count {
				end = i
				break
			}
		}
	}
	if end <= begin {
		return ""
	}
	return string(rs[begin:end])
}

// moraAt is 16e560: the single mora at absolute position index within the
// prosodic word (silent morphs do not count).
func moraAt(pw *PWord, index int) string {
	stop := pw.endMorph()
	sum := 0
	for m := pw.First; m != nil && m != stop; m = m.Next {
		if m.Moras > 0 && m.TagPtr == 0 {
			if index < sum+int(m.Moras) {
				return moraRange(m, index-sum, 1)
			}
			sum += int(m.Moras)
		}
	}
	return ""
}

// endMorph is 115280: the first morph of the following prosodic word.
func (p *PWord) endMorph() *Morph {
	if p.Next == nil {
		return nil
	}
	return p.Next.First
}
