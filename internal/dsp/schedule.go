package dsp

import (
	"fmt"
	"math"
)

// ScheduledChunk selects a contiguous frame interval from the full acoustic
// model output. Start and End are absolute frame offsets, with End exclusive.
type ScheduledChunk struct {
	Start   int
	End     int
	Options ChunkOptions
}

type SpeechSegment struct {
	PrefixSilence int
	SuffixSilence int
	Chunks        []ScheduledChunk
}

// ScheduleSpeech reproduces the default 48 kHz neural synthesis scheduler.
// Boundary indices identify the padding/pause phonemes shared by segments.
// requested contains their original pause lengths in milliseconds (-1 means
// use the predicted duration). These controls come from text analysis, not from
// the waveform or a stored recording.
func ScheduleSpeech(duration []float32, boundaries, requested []int, totalFrames int) ([]SpeechSegment, error) {
	if len(duration) < 2 || len(requested) != len(duration) || totalFrames <= 0 || totalFrames > 200000 ||
		len(boundaries) < 2 || boundaries[0] != 0 || boundaries[len(boundaries)-1] != len(duration)-1 {
		return nil, fmt.Errorf("invalid speech schedule inputs")
	}
	for _, d := range duration {
		if !finite(d) || d < 0 {
			return nil, fmt.Errorf("invalid phoneme duration")
		}
	}
	for i, b := range boundaries {
		if b < 0 || b >= len(duration) || (i > 0 && b <= boundaries[i-1]) {
			return nil, fmt.Errorf("invalid speech boundaries")
		}
	}
	for _, v := range requested {
		if v < -1 || v > 600000 {
			return nil, fmt.Errorf("invalid pause duration")
		}
	}
	const frameSamples = 240
	frames := func(d float32) int { return int(math.Round(float64(d / 5))) }
	pauseSamples := func(i int) int {
		v := requested[i]
		if v < 0 {
			v = int(duration[i])
		}
		return v * frameSamples / 5
	}
	segments := make([]SpeechSegment, 0, len(boundaries)-1)
	begin, cumulative := 0, frames(duration[0])
	for group := 1; group < len(boundaries); group++ {
		first, last := boundaries[group-1], boundaries[group]
		var sum float32
		for _, d := range duration[first : last+1] {
			sum += d
		}
		length := frames(sum)
		startPause, endPause := frames(duration[first]), frames(duration[last])
		cumulative += length
		if startPause > 0 {
			cumulative -= startPause
		}
		if group == len(boundaries)-1 && cumulative != totalFrames {
			length += totalFrames - cumulative
		}
		speech := length - endPause
		if length <= 0 || speech < 0 || begin+length > totalFrames {
			return nil, fmt.Errorf("invalid acoustic frame boundary")
		}
		segment := SpeechSegment{SuffixSilence: max(0, pauseSamples(last)-2*frameSamples)}
		if group == 1 {
			segment.PrefixSilence = max(0, pauseSamples(first)-frameSamples)
		}
		if group == len(boundaries)-1 {
			segment.SuffixSilence += frameSamples
		}
		for offset := 0; offset < speech+1; offset += 100 {
			end := min(begin+offset+110, begin+length)
			lastChunk := end >= begin+speech+1
			trim := max(0, end-begin-speech-1) * frameSamples
			segment.Chunks = append(segment.Chunks, ScheduledChunk{Start: begin + offset, End: end, Options: ChunkOptions{
				FrameSamples: frameSamples, OverlapFrames: 10, Threshold: 0.4,
				First: offset == 0, Last: lastChunk, StartSkip: max(0, startPause-1) * frameSamples,
				FrameBegin: begin + offset, EndTrim: trim,
			}})
			if lastChunk {
				break
			}
		}
		segments = append(segments, segment)
		begin += speech
	}
	return segments, nil
}
