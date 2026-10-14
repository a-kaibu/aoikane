package frontend

import (
	"fmt"
	"strings"
)

// DecodeAccentRecord reads the dictionary's variable-length accent record.
// Code bit 0 links another alternative; bit 7 introduces Down/Moras. Bit 6
// instead stores Up/Down/Moras (kaccent). Bits 2..5 select the concatenation rule.
// The exact interpretation of each rule by phrase analysis is separate.
func DecodeAccentRecord(data []byte) ([]Accent, int, error) {
	var records []Accent
	at := 0
	for len(records) < 256 {
		if at >= len(data) {
			return nil, 0, fmt.Errorf("truncated dictionary accent record")
		}
		code := data[at]
		at++
		a := Accent{Code: code}
		if code&0x80 != 0 {
			if len(records) != 0 && code&0x40 != records[0].Code&0x40 {
				return nil, 0, fmt.Errorf("mixed accent parameter formats are not supported")
			}
			need := 2
			if code&0x40 != 0 {
				need++
			}
			if len(data)-at < need {
				return nil, 0, fmt.Errorf("truncated dictionary accent parameters")
			}
			a.Down, a.Moras = int(data[at]), int(data[at+1])
			if need == 3 {
				v := int(data[at])
				a.Up = &v
				a.Down, a.Moras = int(data[at+1]), int(data[at+2])
			}
			at += need
		}
		records = append(records, a)
		if code&1 == 0 {
			return records, at, nil
		}
	}
	return nil, 0, fmt.Errorf("too many dictionary accent alternatives")
}

// FormatAccentRecord reproduces the native dictionary inspector's text form.
// It is an inspection format, not instructions to synthesize a whole phrase.
func FormatAccentRecord(records []Accent) string {
	if len(records) == 0 {
		return ""
	}
	var b strings.Builder
	if records[0].Code&0x40 != 0 {
		b.WriteString("kaccent=")
	} else {
		b.WriteString("accent=")
	}
	for i, a := range records {
		if i != 0 {
			b.WriteByte(',')
		}
		spoken := a.Code&0x80 != 0
		if spoken {
			if a.Up != nil {
				fmt.Fprintf(&b, "%d-", *a.Up)
			}
			fmt.Fprintf(&b, "%d-%d", a.Down, a.Moras)
		}
		join := a.Code >> 2 & 15
		if join != 0 {
			if spoken {
				b.WriteByte(',')
			}
			b.WriteByte('s')
			b.WriteByte(byte(join) + '0' - 1)
		} else if !spoken {
			b.WriteByte('*')
		}
	}
	return b.String()
}
