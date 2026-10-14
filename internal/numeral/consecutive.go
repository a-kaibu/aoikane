package numeral

import "fmt"

func (r *Rules) ReadConsecutiveDigits(a, b string, last Request) ([]Reading, error) {
	if !r.ConsecutiveDigits[a+"，"+b] || last.Counter == "" {
		return nil, fmt.Errorf("unsupported consecutive digits")
	}
	first := r.digitWord(kanjiDigits[[]rune(a)[0]])
	// Eng digit records carry their kanji key separately.
	key := digitKanji[kanjiDigits[[]rune(a)[0]]]
	if alt, ok := altDigit['E'][key]; ok {
		first.Read, first.Sound = alt.Read, alt.Sound
	}
	second := r.digitWord(kanjiDigits[[]rune(b)[0]])
	last.compoundDigits, last.NoSpecial = true, true
	k := digitKanji[kanjiDigits[[]rune(b)[0]]]
	return r.withCounter(last, []chunk{{Read: first.Read + second.Read, Sound: first.Sound + second.Sound, Key: k, TkKey: k}}, false)
}
