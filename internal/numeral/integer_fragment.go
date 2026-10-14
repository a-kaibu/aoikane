package numeral

import (
	"fmt"
	"math/big"
	"strings"
)

// 桁区切り繰上 (+2n#1u) expands a multi-digit Arabic coefficient and one
// written unit before 100a90. For 10000十 the observed input is １０００００,
// not １００００十. Leading-zero digit groups have their own earlier path.
func expandedUnitInteger(number string) (string, bool) {
	chars := []rune(number)
	if len(chars) < 3 || chars[0] == '０' || !strings.ContainsRune("十百千万億兆京垓", chars[len(chars)-1]) ||
		!fullWidthDigits(string(chars[:len(chars)-1])) {
		return "", false
	}
	var coefficient strings.Builder
	for _, c := range chars[:len(chars)-1] {
		coefficient.WriteRune('0' + c - '０')
	}
	value, _ := new(big.Int).SetString(coefficient.String(), 10)
	unit := chars[len(chars)-1]
	if scale, small := smallUnits[unit]; small {
		value.Mul(value, big.NewInt(scale))
	} else {
		for index, large := range bigUnits {
			if unit == large {
				value.Mul(value, new(big.Int).Exp(big.NewInt(10000), big.NewInt(int64(index+1)), nil))
				break
			}
		}
	}
	var expanded strings.Builder
	for _, c := range value.String() {
		expanded.WriteRune('０' + c - '0')
	}
	return expanded.String(), true
}

// 100a90's leading-zero digit groups are paired before a final large unit.
// They retain the digit reader's accents rather than ordinary sound lookup.
func pairedZeroLargeUnit(number string) bool {
	chars := []rune(number)
	return len(chars) >= 3 && strings.ContainsRune(string(bigUnits), chars[len(chars)-1]) &&
		leadingZeroDigitRun(string(chars[:len(chars)-1]))
}

// integerFragmentSound reconstructs 10b010's right-to-left positional
// expansion. Written units reset the position and survive literally; they
// are not added together as values. Thus 十十 remains 十十 and １百 becomes
// 一百, while １００ expands to 百.
func (r *Rules) integerFragmentSound(number string) (string, error) {
	chars := []rune(number)
	var zeros string
	for len(chars) > 0 && (chars[0] == '０' || chars[0] == '〇') {
		zeros += r.digitWord(0).Sound
		chars = chars[1:]
	}
	position, large := 0, 0
	pendingLarge := ""
	digit := false
	canonical := ""
	for i := len(chars) - 1; i >= 0; i-- {
		c := chars[i]
		if unit, ok := smallUnits[c]; ok {
			switch unit {
			case 10:
				position = 1
			case 100:
				position = 2
			case 1000:
				position = 3
			}
			digit = false
			canonical = string(c) + canonical
			continue
		}
		if index := strings.IndexRune(string(bigUnits), c); index >= 0 {
			large = len([]rune(string(bigUnits)[:index])) + 1
			position, digit = 0, false
			canonical = string(c) + canonical
			continue
		}
		d, ok := kanjiDigits[c]
		if !ok {
			return "", fmt.Errorf("unsupported numeric fragment %q", number)
		}
		if !digit {
			if d != 0 || c == '零' {
				key := digitKanji[d]
				if c == '零' {
					key = "零"
				}
				if zeros != "" && d == 1 && position > 0 && large == 0 {
					key = ""
				}
				canonical = key + canonical
			}
			digit = true
			continue
		}
		position++
		var suffix string
		if position >= 4 {
			position = 0
			large++
			if large > len(bigUnits) {
				return "", fmt.Errorf("numeric fragment exceeds large-unit table: %q", number)
			}
			pendingLarge = string(bigUnits[large-1])
		} else if position > 0 {
			suffix = string([]rune("十百千")[position-1])
		}
		if d == 0 {
			continue
		}
		key := digitKanji[d]
		if d == 1 && (position == 1 || position == 2 || position == 3 && large == 0) {
			key = ""
		}
		canonical = key + suffix + pendingLarge + canonical
		pendingLarge = ""
	}
	var sound string
	canon := []rune(canonical)
	for i := 0; i < len(canon); i++ {
		c := canon[i]
		if d, ok := kanjiDigits[c]; ok {
			if i+1 < len(canon) {
				if unit, small := smallUnits[canon[i+1]]; small && d > 1 {
					for _, part := range r.groupChunks(d * int(unit)) {
						sound += part.Sound
					}
					i++
					continue
				}
				if d == 1 && canon[i+1] == '千' {
					sound += "イッセン"
					i++
					continue
				}
			}
			sound += r.Digits[string(c)].Sound
			continue
		}
		unit := string(c)
		if strings.ContainsRune(string(bigUnits), c) {
			sound = attachLargeUnit(chunk{Sound: sound}, unit).Sound
		}
		sound += r.Units[unit].Sound
	}
	return zeros + sound, nil
}
