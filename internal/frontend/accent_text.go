package frontend

import "fmt"

// AccentTextRecord is one 1590b0 record. Field names stay positional because
// callers interpret the values differently; word cleanup uses Second.
type AccentTextRecord struct {
	First   int32 `json:"first"`
	Second  int32 `json:"second"`
	Third   int32 `json:"third"`
	Speaker int32 `json:"speaker"`
	Next    int   `json:"next"`
}

func ParseAccentText(text string, start int) (AccentTextRecord, error) {
	out := AccentTextRecord{Speaker: -1}
	if len(text) > 4096 || start < 0 || start > len(text) {
		return out, fmt.Errorf("invalid accent text scope")
	}
	for _, c := range text {
		if c == 0 || c > 127 {
			return out, fmt.Errorf("accent text requires non-NUL ASCII")
		}
	}
	at := func(i int) byte {
		if i >= len(text) {
			return 0
		}
		return text[i]
	}
	digit := func(c byte) bool { return c >= '0' && c <= '9' }
	atoi := func(i int) (int32, error) {
		for at(i) == ' ' || at(i) == '\t' || at(i) == '\n' || at(i) == '\r' || at(i) == '\v' || at(i) == '\f' {
			i++
		}
		sign := int64(1)
		if at(i) == '-' || at(i) == '+' {
			if at(i) == '-' {
				sign = -1
			}
			i++
		}
		var value int64
		for digit(at(i)) {
			value = value*10 + int64(at(i)-'0')
			if value > 2147483648 {
				value = 2147483648
			}
			i++
		}
		value *= sign
		if value > 2147483647 {
			value = 2147483647
		}
		return int32(value), nil
	}
	if at(0) == '*' {
		out.First = 2
		return out, nil
	}
	i := start
	if at(i) == 'p' || at(i) == 's' {
		i++
		for i < len(text)-1 {
			if at(i) == ',' && digit(at(i+1)) {
				i++
				break
			}
			i++
		}
	}
	begin := i
	for i < len(text) {
		c := at(i)
		i++
		if c == ',' {
			break
		}
	}
	end := i
	if at(i) == 'p' || at(i) == 's' {
		for i < len(text)-1 {
			if i == 0 || at(i-1) == ',' {
				if digit(at(i)) {
					break
				}
				if at(i) == 's' {
					v, err := atoi(i + 1)
					if err != nil {
						return out, err
					}
					out.Speaker = v
				}
			}
			i++
		}
	}
	if i < len(text) && at(i) != 0 {
		out.Next = i
	}
	if begin < end {
		v, err := atoi(begin)
		if err != nil {
			return out, err
		}
		out.First = v
		for j := begin; j < end; j++ {
			if at(j) == '-' {
				v, err := atoi(j + 1)
				if err != nil {
					return out, err
				}
				out.Second = v
				for k := j + 1; k < end; k++ {
					if at(k) == '-' {
						v, err := atoi(k + 1)
						if err != nil {
							return out, err
						}
						out.Third = v
						break
					}
				}
				break
			}
		}
	}
	return out, nil
}
