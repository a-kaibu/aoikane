package synthesis

import (
	"fmt"
	"math"

	"github.com/a-kaibu/aoikane/internal/frontend"
)

// Controls are voice-specific style coordinates and user-selected controls.
// They contain no phoneme, acoustic, or recorded waveform data.
type Controls struct {
	Style           [8]float32 `json:"style"`
	Speed           float32    `json:"speed"`
	Pitch           float32    `json:"pitch"`
	Intonation      float32    `json:"intonation"`
	Loudness        float32    `json:"loudness"`
	BeginPauseMS    int        `json:"begin_pause_ms"`
	EndPauseMS      int        `json:"end_pause_ms"`
	InternalPauseMS [3]int     `json:"internal_pause_ms"`
}

// FromPhones constructs the neural input contract from the reconstructed
// frontend. Text reading and phrase accent selection remain upstream.
func FromPhones(sequence frontend.PhoneSequence, controls Controls) (Input, error) {
	n := len(sequence.Phones)
	if n < 2 || len(sequence.Articulation) != n || len(sequence.Phoneme) != n {
		return Input{}, fmt.Errorf("invalid frontend phone sequence")
	}
	for _, ms := range append([]int{controls.BeginPauseMS, controls.EndPauseMS}, controls.InternalPauseMS[:]...) {
		if ms < 0 || ms > 600000 {
			return Input{}, fmt.Errorf("invalid pause duration")
		}
	}
	for _, v := range append(controls.Style[:], controls.Speed, controls.Pitch, controls.Intonation, controls.Loudness) {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return Input{}, fmt.Errorf("non-finite synthesis control")
		}
	}
	out := Input{
		Phoneme:        append([]int64(nil), sequence.Phoneme...),
		Accent:         append([]int64(nil), sequence.Accent...),
		Punctuation:    append([]int64(nil), sequence.Punctuation...),
		MoraPosition:   append([]float32(nil), sequence.MoraPosition...),
		PhrasePosition: append([]float32(nil), sequence.PhrasePosition...),
		Boundaries:     append([]int(nil), sequence.Boundaries...),
		Art:            make([]float32, 32*n), Style: make([]float32, 8*n),
		Speed: make([]float32, n), Pitch: make([]float32, n),
		Intonation: make([]float32, n), Loudness: make([]float32, n),
		Duration: make([]float32, n), F0: make([]float32, n),
		Power: make([]float32, n), RequestedPauseMS: make([]int, n),
	}
	for i, p := range sequence.Phones {
		out.Phoneme[i] = int64(p.ID)
		for channel := 0; channel < 32; channel++ {
			out.Art[channel*n+i] = float32((sequence.Articulation[i] >> channel) & 1)
		}
		for channel, value := range controls.Style {
			out.Style[channel*n+i] = value
		}
		out.Speed[i], out.Pitch[i] = controls.Speed, controls.Pitch
		out.Intonation[i], out.Loudness[i] = controls.Intonation, controls.Loudness
		out.Duration[i], out.F0[i], out.Power[i] = -1, -1, -1
		out.RequestedPauseMS[i] = -1
		if p.ID == 0 {
			// Native pause records specify 100 ms to the duration model;
			// the separate render schedule applies the requested duration.
			out.Duration[i] = 100
			switch {
			case i == 0:
				out.RequestedPauseMS[i] = controls.BeginPauseMS
			case i == n-1:
				out.RequestedPauseMS[i] = controls.EndPauseMS
			default:
				index, ok := map[byte]int{'B': 0, 'P': 1, '-': 2}[p.Label[3]]
				if !ok {
					return Input{}, fmt.Errorf("unsupported internal pause %q", p.Label)
				}
				out.RequestedPauseMS[i] = controls.InternalPauseMS[index]
			}
		}
	}
	if _, err := out.tensors(); err != nil {
		return Input{}, err
	}
	return out, nil
}
