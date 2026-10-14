package frontend

// EnglishDecisionFeatures contains the normal (non-diagnostic) feature vectors
// built by 1a81f0 and 1a704f..1a7351. Integer slot 31 is outside the initialized
// native context and is not consumed by the inspected English trees.
type EnglishDecisionFeatures struct {
	Strings  [30]string
	Integers [32]int32
}

func BuildEnglishDecisionFeatures(units []PreparedEnglishSegment, index, optionA, optionB int) (EnglishDecisionFeatures, error) {
	context, err := BuildEnglishConditionContext(units, index, optionA, optionB)
	if err != nil {
		return EnglishDecisionFeatures{}, err
	}
	var out EnglishDecisionFeatures
	for i, u := range context.Units {
		s := 4 + i*5
		out.Strings[s], out.Strings[s+1], out.Strings[s+2], out.Strings[s+4] = u.Primary, u.VowelPrefix, u.VowelTail, u.Combined
		out.Integers[s+3] = int32(u.Mask)
	}
	out.Strings[27] = context.LastCharacter
	out.Integers[25], out.Integers[26] = int32(context.Following), int32(index)
	out.Integers[28], out.Integers[29], out.Integers[30] = int32(optionA), int32(optionB), 8
	next := "_"
	if index+1 < len(units) {
		u := units[index+1]
		for _, s := range []string{u.Primary, u.VowelPrefix, u.VowelTail} {
			if s != "" {
				next = s[:1]
				break
			}
		}
	}
	u := units[index]
	tail, prefix := next, next
	if u.VowelTail != "" {
		tail = u.VowelTail[:1]
	}
	prefix = tail
	if u.VowelPrefix != "" {
		prefix = u.VowelPrefix[:1]
	}
	out.Strings[19], out.Strings[20], out.Strings[21] = prefix, tail, next
	previous := "_"
	if index > 0 {
		p := units[index-1]
		for _, s := range []string{p.VowelTail, p.VowelPrefix, p.Primary} {
			if s != "" {
				previous = s[len(s)-1:]
				break
			}
		}
	}
	primaryLast := previous
	if u.Primary != "" {
		primaryLast = u.Primary[len(u.Primary)-1:]
	}
	prefixLast := primaryLast
	if u.VowelPrefix != "" {
		prefixLast = u.VowelPrefix[len(u.VowelPrefix)-1:]
	}
	out.Strings[22], out.Strings[23], out.Strings[24] = previous, primaryLast, prefixLast
	return out, nil
}
