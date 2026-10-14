package phrase

import "strings"

// Connection rule identifiers returned by ParseRule (native 1586b0/158b30).
const (
	RuleP1 = iota
	RuleP2
	RuleP3
	RuleP4
	RuleC1
	RuleC2
	RuleC3
	RuleC4
	RuleF1
	RuleF2
	RuleF3
	RuleF4
	RuleF5
	RuleF6
	RuleS1
	RuleS2
	RuleS3
	RuleK1
	RuleK2
	RuleK3
	RuleK4
	RuleK5
	RuleK6
	RuleK7
	RuleK8
	RuleA1
	RuleNone   // 0x1a: "*" or unparsable
	RuleAbsent // 0x1b: no entry for the requested class
)

// classNames are the preceding-word conditions of a connection text
// (native table at 18038e7c0, index 1..5).
var classNames = [...]string{"", "名詞", "動詞", "形容詞", "特殊助動詞", "連体修飾節"}

// atoi follows the C library: optional leading white space, optional sign,
// then digits. A string without digits yields 0.
func atoi(s string) int {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] >= 9 && s[i] <= 13) {
		i++
	}
	neg := false
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		neg = s[i] == '-'
		i++
	}
	n := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		n = n*10 + int(s[i]-'0')
		i++
	}
	if neg {
		return -n
	}
	return n
}

func digit(c byte) bool { return c >= '0' && c <= '9' }

// at returns s[i] or 0 past the end, like reading a NUL-terminated string.
func at(s string, i int) byte {
	if i >= 0 && i < len(s) {
		return s[i]
	}
	return 0
}

// ParseRule reconstructs 1586b0: it selects the connection rule of a
// candidate for the preceding word's class (1..5; 0 selects the unconditional
// default) and returns the rule with the accent values. First and Second are
// the rise and fall of the candidate's accent text, as modified by the rule
// ("*N", "@N", or restored by "else"). prevMoras feeds the eq/ge/le tests.
func ParseRule(accent, conn string, class int, prevMoras int32) (rule int, first, second int) {
	first, second = -9999, -9999
	var rest string
	if class == 0 {
		n := len(conn)
		if n == 0 {
			return RuleNone, first, second
		}
		if conn[0] >= 0x80 { // starts with a class name: look for an unconditional entry
			found := false
			for i := 1; i < n; i++ {
				if conn[i] == ',' && at(conn, i+1) < 0x80 {
					rest, found = conn[i+1:], true
					break
				}
			}
			if !found {
				return RuleNone, first, second
			}
		} else {
			rest = conn
		}
	} else {
		pos := strings.Index(conn, classNames[class])
		if pos < 0 || at(conn, pos+len(classNames[class])) != '%' {
			return RuleAbsent, first, second
		}
		rest = conn[pos+len(classNames[class])+1:]
	}
	first, second = parseAccentPair(accent)
	return parseRuleText(rest, &first, &second, prevMoras), first, second
}

// parseAccentPair extracts the first two numbers of a candidate accent text
// ("2-0-4", "p,2-4", "s3,2-4"). A leading "*" means flat (first = 2).
func parseAccentPair(acc string) (first, second int) {
	if at(acc, 0) == '*' {
		return 2, 0
	}
	n := len(acc)
	start := 0
	if c := at(acc, 0); c == 'p' || c == 's' {
		start = 1
		if 1 < n-1 {
			for j := 2; ; j++ {
				start = j
				if at(acc, j-1) == ',' && digit(at(acc, j)) {
					break
				}
				if j >= n-1 {
					break
				}
			}
		}
	}
	end := start
	if start < n {
		for {
			end++
			if at(acc, end-1) == ',' || end >= n {
				break
			}
		}
	}
	if start < end {
		first = atoi(acc[start:])
		for i := start; i < end; i++ {
			if at(acc, i) == '-' {
				second = atoi(acc[i+1:])
				break
			}
		}
	}
	return first, second
}

// parseRuleText reconstructs 158b30: tokens of one connection entry.
func parseRuleText(text string, p1, p2 *int, prevMoras int32) int {
	n := len(text)
	save2, save1 := *p2, *p1
	rule := RuleNone
	number := func(i int) (int, int) {
		v := 0
		for digit(at(text, i)) {
			v = v*10 + int(text[i]-'0')
			i++
		}
		return v, i
	}
	for i := 0; n != 0 && i < n; {
		c := text[i]
		switch c {
		case 0, ',', '|':
			return rule
		case '*', '@':
			target := p1
			if c == '@' {
				target = p2
			}
			if n < i+2 || !digit(text[i+1]) && text[i+1] != '-' {
				i++
				continue
			}
			i++
			*target = atoi(text[i:])
			for i < n && (digit(text[i]) || text[i] == '-') {
				i++
			}
			continue
		case 'e':
			if i+4 < n && text[i+1] == 'l' && text[i+2] == 's' && text[i+3] == 'e' {
				*p2, *p1, rule = save2, save1, RuleNone
				i += 3
				continue
			}
			if n <= i+1 || text[i+1] != 'q' {
				i++
				continue
			}
			v, j := number(i + 2)
			i = j
			if v >= 0 && int(prevMoras) == v {
				return rule
			}
			continue
		case 'g', 'l':
			want := byte('e')
			if n <= i+1 || text[i+1] != want {
				i++
				continue
			}
			v, j := number(i + 2)
			i = j
			if c == 'g' && v >= 0 && v <= int(prevMoras) || c == 'l' && v >= 0 && int(prevMoras) <= v {
				return rule
			}
			continue
		}
		if c == 'C' && at(text, i+1) == 'F' {
			i++
		}
		switch text[i] {
		case 'A':
			if at(text, i+1) != '1' {
				return rule
			}
			rule, i = RuleA1, i+2
		case 'C':
			d := at(text, i+1)
			if d < '1' || d > '4' {
				return rule
			}
			rule, i = RuleC1+int(d-'1'), i+2
		case 'F':
			d := at(text, i+1)
			if d < '1' || d > '6' {
				return rule
			}
			rule, i = RuleF1+int(d-'1'), i+2
		case 'K':
			d := at(text, i+1)
			if d < '1' || d > '8' {
				return rule
			}
			rule, i = RuleK1+int(d-'1'), i+2
		case 'P':
			d := at(text, i+1)
			if d < '1' || d > '4' {
				return rule
			}
			rule, i = RuleP1+int(d-'1'), i+2
		case 'S':
			d := at(text, i+1)
			if d < '1' || d > '3' {
				return rule
			}
			rule, i = RuleS1+int(d-'1'), i+2
		case '*':
			return RuleNone
		default:
			return rule
		}
	}
	return rule
}
