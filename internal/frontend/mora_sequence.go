package frontend

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// MoraUnit maps one pronunciation unit to the inspected phone definitions.
// Long identifies an explicit ー. Contextual labels, accent, devoicing and
// the preceding vowel's length bitmap are applied by later stages.
type MoraUnit struct {
	Text         string `json:"text"`
	PhoneIndices []int  `json:"phone_indices"`
	Long         bool   `json:"long,omitempty"`
}

// RVA 0x22100 extends the last non-pause vowel, N or Q. A leading
// long mark has no phone, but remains a mora for accent/position counting.
func (e *MoraEncoder) longPhoneIndices(previous []int) []int {
	if len(previous) == 0 {
		return nil
	}
	index := previous[len(previous)-1]
	id := e.Phones[index].ID
	if (id >= 1 && id <= 5) || id == 50 || id == 51 {
		return []int{index}
	}
	return nil
}

// EncodePronunciation segments a complete kana pronunciation. It preserves
// native lookup priority and does not choose a reading or accent alternative.
func (e *MoraEncoder) EncodePronunciation(text string, expanded bool) ([]MoraUnit, error) {
	if !utf8.ValidString(text) || len(text) > 1<<20 {
		return nil, fmt.Errorf("pronunciation must be valid UTF-8 and at most 1 MiB")
	}
	units := make([]MoraUnit, 0)
	var previous []int
	for at := 0; at < len(text); {
		if strings.HasPrefix(text[at:], "ー") {
			units = append(units, MoraUnit{Text: "ー", PhoneIndices: e.longPhoneIndices(previous), Long: true})
			at += len("ー")
			continue
		}
		phones, length := e.LookupPrefix(text[at:], expanded)
		if length == 0 {
			return nil, fmt.Errorf("unsupported pronunciation at byte %d", at)
		}
		units = append(units, MoraUnit{Text: text[at : at+length], PhoneIndices: phones})
		previous = phones
		at += length
	}
	return units, nil
}
