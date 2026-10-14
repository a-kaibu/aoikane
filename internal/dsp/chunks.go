package dsp

import (
	"fmt"
	"math"
)

// ChunkOptions describes the native stream joiner's frame and boundary controls.
// StartSkip and EndTrim are sample counts derived by the synthesis scheduler.
type ChunkOptions struct {
	FrameSamples  int     `json:"frame_samples"`
	OverlapFrames int     `json:"overlap_frames"`
	Threshold     float32 `json:"similarity_threshold"`
	First         bool    `json:"first"`
	Last          bool    `json:"last"`
	StartSkip     int     `json:"initial_skip"`
	CarriedSkip   int     `json:"carried_skip"`
	SkipFrames    int     `json:"skip_frames"`
	FrameBegin    int     `json:"frame_begin"`
	EndTrim       int     `json:"end_trim"`
}

type JoinedChunk struct {
	PCM          []int16
	Tail         []float32
	CarriedSkip  int
	CandidateLag int
	AppliedLag   int
}

// PearsonCorrelation uses the native float32 accumulation and normalization.
// Silence and constant vectors have zero similarity when the denominator is zero.
func PearsonCorrelation(a, b []float32) (float32, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("similarity vector lengths differ")
	}
	var sa, sb, aa, bb, ab float32
	for i, x := range a {
		y := b[i]
		if !finite(x) || !finite(y) {
			return 0, fmt.Errorf("non-finite similarity input")
		}
		sa += x
		sb += y
		aa += x * x
		bb += y * y
		ab += x * y
	}
	n := float32(len(a))
	va, vb := n*aa-sa*sa, n*bb-sb*sb
	denom := float32(math.Sqrt(float64(va * vb)))
	if denom == 0 {
		return 0, nil
	}
	return (n*ab - sa*sb) / denom, nil
}

func linearRamp(n int) []float32 {
	v := make([]float32, n)
	if n < 2 {
		return v
	}
	step := float32(1) / float32(n-1)
	for i := 1; i < n-1; i++ {
		v[i] = v[i-1] + step
	}
	v[n-1] = 1
	return v
}

func pcmSample(v float32) int16 {
	return int16(int32(math.Round(float64(v * 32767))))
}

// JoinChunk ports the original phase alignment, fade and PCM conversion.
// previous is the un-emitted tail from the prior chunk. Sentence pauses are
// supplied by the caller; they are not invented from audio or reference cases.
func JoinChunk(wave, previous []float32, o ChunkOptions) (JoinedChunk, error) {
	var out JoinedChunk
	if o.FrameSamples != 240 || o.OverlapFrames < 0 || o.OverlapFrames > 100 ||
		o.StartSkip < 0 || o.CarriedSkip < 0 || o.EndTrim < 0 || o.EndTrim > len(wave) ||
		o.SkipFrames < 0 || o.FrameBegin < 0 || !finite(o.Threshold) {
		return out, fmt.Errorf("unsupported chunk controls")
	}
	for _, v := range wave {
		if !finite(v) {
			return out, fmt.Errorf("non-finite chunk waveform")
		}
	}
	for _, v := range previous {
		if !finite(v) {
			return out, fmt.Errorf("non-finite previous waveform")
		}
	}
	overlap := o.FrameSamples * o.OverlapFrames
	if !o.First && o.OverlapFrames >= 2 {
		center := overlap / 2
		lo, hi := center-o.FrameSamples, center+o.FrameSamples
		if len(previous) < hi || len(wave) < hi {
			return out, fmt.Errorf("chunk alignment window is truncated")
		}
		c, err := CrossCorrelation(wave[lo:hi], previous[lo:hi])
		if err != nil {
			return out, err
		}
		peak := 0
		for i := 1; i < len(c); i++ {
			if c[i] > c[peak] {
				peak = i
			}
		}
		lag := peak - (hi - lo - 1)
		out.CandidateLag = lag
		if lag > 0 {
			if len(wave) < lag+overlap || len(previous) < overlap {
				return out, fmt.Errorf("positive alignment window is truncated")
			}
			score, err := PearsonCorrelation(wave[lag:lag+overlap], previous[:overlap])
			if err != nil {
				return out, err
			}
			if score > o.Threshold {
				// The native erase begins one sample before the selected lag.
				wave = wave[lag-1:]
				out.AppliedLag = lag - 1
			}
		} else if lag < 0 {
			length := overlap + lag
			if length < 0 || len(wave) < length || len(previous) < overlap {
				return out, fmt.Errorf("negative alignment window is truncated")
			}
			score, err := PearsonCorrelation(wave[:length], previous[-lag:overlap])
			if err != nil {
				return out, err
			}
			if score > o.Threshold {
				out.AppliedLag = lag
			}
		}
	}
	usable := len(wave) - o.EndTrim
	if usable < 0 {
		return out, fmt.Errorf("alignment consumed the usable waveform")
	}
	appendPCM := func(values []float32) {
		for _, v := range values {
			out.PCM = append(out.PCM, pcmSample(v))
		}
	}
	cursor := o.CarriedSkip
	if o.First {
		cursor = o.StartSkip
		if o.SkipFrames > 0 {
			cursor = max(cursor, (o.SkipFrames-o.FrameBegin)*o.FrameSamples)
		}
		if !o.Last && usable >= overlap && cursor > usable-overlap {
			out.CarriedSkip = cursor - usable + overlap
			cursor = usable - overlap
		}
	} else if usable >= overlap && overlap > 0 {
		if len(previous) < overlap {
			return out, fmt.Errorf("previous overlap is truncated")
		}
		if out.AppliedLag < 0 {
			pad := -out.AppliedLag
			length := overlap - pad
			if cursor > length {
				return out, fmt.Errorf("carried skip exceeds aligned fade")
			}
			appendPCM(previous[:pad])
			ramp := linearRamp(length)
			for ; cursor < length; cursor++ {
				v := wave[cursor]*ramp[cursor] + previous[pad+cursor]*ramp[length-1-cursor]
				out.PCM = append(out.PCM, pcmSample(v))
			}
		} else {
			ramp := linearRamp(overlap)
			for ; cursor < overlap; cursor++ {
				v := wave[cursor]*ramp[cursor] + previous[cursor]*ramp[overlap-1-cursor]
				out.PCM = append(out.PCM, pcmSample(v))
			}
		}
	}
	end := usable
	if !o.Last {
		end = usable - overlap
	}
	if cursor < 0 || cursor > usable {
		return out, fmt.Errorf("chunk skip exceeds usable waveform")
	}
	if cursor < end {
		appendPCM(wave[cursor:end])
		cursor = end
	}
	if !o.Last {
		out.Tail = append([]float32(nil), wave[cursor:usable]...)
	}
	return out, nil
}
