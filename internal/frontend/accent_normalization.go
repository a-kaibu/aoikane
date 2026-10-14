package frontend

import "fmt"

// NormalizeAccent fills the native rise position for ordinary dictionary
// accents. Explicit kaccent records retain their stored Up/Down/Moras.
// This matches 1593c0's accent-property conversion; it does not merge phrases.
func NormalizeAccent(records []Accent) ([]Accent, error) {
	if len(records) == 0 {
		return nil, fmt.Errorf("missing accent records")
	}
	// Native normalization treats a leading '*' as the entire accent value,
	// even if the dictionary inspector printed later linked alternatives.
	if records[0].Code&0xc0 == 0 && records[0].Code>>2&15 == 0 {
		a := records[0]
		a.Code = a.Code&^1 | 0x40
		return []Accent{a}, nil
	}
	out := make([]Accent, len(records))
	for i, a := range records {
		// Join 11 is rendered as s: by the binary inspector. The colon is
		// ambiguous with the property separator and native normalization
		// rejects that inspection form, rather than silently accepting it.
		if a.Code>>2&15 == 11 {
			return nil, fmt.Errorf("accent join 11 has an ambiguous native property representation")
		}
		if a.Code&0x80 != 0 {
			if a.Code&0x40 != 0 {
				if a.Up == nil {
					return nil, fmt.Errorf("explicit kana accent has no rise position")
				}
				up := *a.Up
				a.Up = &up
			} else {
				up := 2
				if a.Moras == 1 {
					up = 0
					if a.Down == 1 {
						up = 1
					}
				} else if a.Down == 1 {
					up = 1
				}
				a.Up = &up
			}
		}
		a.Code |= 0x40
		out[i] = a
	}
	return out, nil
}
