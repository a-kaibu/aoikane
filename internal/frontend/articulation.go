package frontend

import "fmt"

// ModelArticulation expands the stored phone bitmap with contextual length,
// voicing, repetition and pause flags. Its result is the bitmap
// used by the model, before expansion to the 32 channel-major float channels.
// Prosody and devoicing must already have populated Info and Label.
func (e *MoraEncoder) ModelArticulation(phones []Allophone) ([]uint32, error) {
	if len(e.Phones) < 52 {
		return nil, fmt.Errorf("missing base articulation definitions")
	}
	result := make([]uint32, len(phones))
	for i, p := range phones {
		if p.ID < 0 || p.ID >= 52 || len(p.Label) != 4 {
			return nil, fmt.Errorf("invalid articulation phone at %d", i)
		}
		art := e.Phones[p.ID].Art
		if p.ID == 0 {
			// RVA 0x2148e/0x214f3 updates the stored bitmap when the
			// voicing pass clears bit 0x4000, including internal pauses.
			if p.Info&0x4000 == 0 {
				art = art&^16 | 8
			}
			switch p.Label[3] {
			case 'S':
				art |= 1 << 29
			case 'F', 'N', 'H', 'R', 'A':
				art |= 1 << 30
			case 'B', 'P', '-':
				art |= 1 << 31
			default:
				if i != 0 {
					return nil, fmt.Errorf("unsupported pause label %q", p.Label)
				}
				// A leading long mora can annotate the padding's label;
				// its already stored sentence-start bitmap is retained.
				art |= 1 << 29
			}
		} else {
			if (p.ID == 50 || p.ID == 51) && p.Info&0x10 != 0 {
				art |= 1 << 24
			}
			// The moraic nasal can receive a vowel-like final Info after the
			// bitmap was made. It retains the nasal's base articulation.
			vowel := p.ID >= 1 && p.ID <= 5 && p.Info&0x8000 != 0
			if vowel {
				if p.Info&0x2000 != 0 {
					art |= 1 << 24
				}
				if p.Info&0x1000 != 0 {
					return nil, fmt.Errorf("nasal vowel articulation is not reconstructed at %d", i)
				}
				if p.Info&0x4000 != 0 {
					art = art&^8 | 16
				}
			} else if p.ID != 50 && i+1 < len(phones) && p.ID == phones[i+1].ID {
				art |= 1 << 26
			}
		}
		result[i] = art
	}
	return result, nil
}
