package frontend

import (
	"fmt"
	"strings"
)

// ReadingAccent is the cached reading state at offsets 8..14 of the native
// pronunciation container. Its mora count comes from the selected pronunciation,
// rather than the third value stored in the accent text.
type ReadingAccent struct {
	Up      int32 `json:"up"`
	Down    int32 `json:"down"`
	Moras   int32 `json:"moras"`
	Speaker int32 `json:"speaker"`
}

// CountAccentMoras reconstructs 1eb390 for valid non-NUL CP932 strings.
// Every pair recognized by 1eb5d0 counts except its small-kana/handakuten
// table. Unrecognized bytes advance individually, including private-use CP932
// pairs. It is not phoneme segmentation or the Windows decoder's character count.
func CountAccentMoras(pronunciation string, codec *CP932Codec) (int32, error) {
	if codec == nil {
		return 0, fmt.Errorf("missing CP932 codec")
	}
	if strings.IndexByte(pronunciation, 0) >= 0 {
		return 0, fmt.Errorf("reading contains NUL")
	}
	data, err := codec.Encode(pronunciation)
	if err != nil {
		return 0, err
	}
	if len(data) > 65536 {
		return 0, fmt.Errorf("reading exceeds mora counter limit")
	}
	var count int32
	for i := 0; i < len(data); i++ {
		first := data[i]
		second := byte(0)
		if i+1 < len(data) {
			second = data[i+1]
		}
		if !nativeReadingPair(first, second) {
			continue
		}
		i++
		code := uint16(first)<<8 | uint16(data[i])
		switch code {
		case 0x8340, 0x8342, 0x8344, 0x8346, 0x8348, 0x8383, 0x8385, 0x8387, 0x838e, 0x8395, 0x8396, 0x814b:
		default:
			count++
		}
	}
	return count, nil
}

// nativeReadingPair reconstructs the engine's recognized two-byte alphabet
// (1eb5d0), which is narrower than Windows CP932.
func nativeReadingPair(first, second byte) bool {
	between := func(low, high byte) bool { return second >= low && second <= high }
	common := between(0x40, 0xfc) && second != 0x7f
	switch first {
	case 0x81:
		if between(0x40, 0xac) && second != 0x7f {
			return true
		}
		index := byte(second + 0x48)
		return index < 64 && (uint64(0xff01fffc007f00ff)>>index)&1 != 0 || second == 0xfc
	case 0x82:
		return between(0x4f, 0x58) || between(0x60, 0x79) || between(0x81, 0x9a) || between(0x9f, 0xf1)
	case 0x83:
		return between(0x40, 0x96) && second != 0x7f || between(0x9f, 0xb6) || between(0xbf, 0xd6)
	case 0x84:
		return between(0x40, 0x60) || between(0x70, 0x91) && second != 0x7f || between(0x9f, 0xbe)
	case 0x87:
		return between(0x40, 0x75) && second != 0x5e || between(0x7e, 0x9c) && second != 0x7f
	case 0x88:
		return between(0x9f, 0xfc)
	case 0x98:
		return between(0x40, 0x72) || between(0x9f, 0xfc)
	case 0xea:
		return between(0x40, 0xa4) && second != 0x7f
	case 0xed, 0xfa, 0xfb:
		return common
	case 0xee:
		return between(0x40, 0xec) && second != 0x7f || between(0xef, 0xfc)
	case 0xfc:
		return between(0x40, 0x4b)
	default:
		return (first >= 0x89 && first <= 0x97 || first >= 0x99 && first <= 0x9f || first >= 0xe0 && first <= 0xe9) && common
	}
}

// UpdateReadingAccent reconstructs 177380. A missing selected record preserves
// the previous numeric accent fields before recalculating the empty reading's
// count and clamping. A selected record replaces them with ParseAccentText.
func UpdateReadingAccent(previous ReadingAccent, selected *PronunciationCandidate, codec *CP932Codec) (ReadingAccent, error) {
	out := previous
	pronunciation := ""
	if selected != nil {
		parsed, err := ParseAccentText(selected.Accent, 0)
		if err != nil {
			return out, err
		}
		out.Up, out.Down, out.Speaker = parsed.First, parsed.Second, parsed.Speaker
		pronunciation = selected.Pronunciation
	}
	count, err := CountAccentMoras(pronunciation, codec)
	if err != nil {
		return out, err
	}
	out.Moras = count
	if out.Down > out.Moras {
		out.Down = out.Moras
	}
	if out.Down != 0 {
		if out.Up > out.Down {
			out.Up = out.Down
		} else if out.Up == 0 {
			out.Up = 1
		}
	}
	return out, nil
}
