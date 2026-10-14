// Compare the portable joiner with native PCM emitted for every observed chunk.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/a-kaibu/aoikane/internal/dsp"
	"github.com/a-kaibu/aoikane/internal/engine"
)

type tensor struct {
	ID      int      `json:"id"`
	Shape   []int    `json:"shape"`
	Members []tensor `json:"members"`
}
type event struct {
	dsp.ChunkOptions
	Event      string `json:"event"`
	Run        int    `json:"run"`
	Wave       tensor `json:"wave"`
	Previous   tensor `json:"previous"`
	Prefix     tensor `json:"prefix"`
	Output     tensor `json:"output"`
	Boundaries []int  `json:"boundaries"`
	Phones     []struct {
		Requested int `json:"requested_ms"`
	} `json:"phones"`
}

func main() {
	root := flag.String("trace", "private/schedule-trace", "native chunk and scheduler trace")
	flag.Parse()
	if err := run(*root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(root string) error {
	b, err := os.ReadFile(filepath.Join(root, "events.json"))
	if err != nil {
		return err
	}
	var events []event
	if err := json.Unmarshal(b, &events); err != nil {
		return err
	}
	outputs := map[int]tensor{}
	for _, e := range events {
		if e.Event == "chunk_join_result" {
			outputs[e.Run] = e.Output
		}
	}
	load := func(t tensor, size int) ([]byte, error) {
		if t.ID == 0 || len(t.Shape) == 0 || len(t.Shape) > 3 {
			return nil, fmt.Errorf("invalid chunk tensor")
		}
		b, err := os.ReadFile(filepath.Join(root, fmt.Sprintf("model-%d.bin", t.ID)))
		if err != nil {
			return nil, err
		}
		count := 1
		for _, d := range t.Shape {
			if d < 0 || (d > 0 && count > 2000000/d) {
				return nil, fmt.Errorf("invalid tensor shape")
			}
			count *= d
		}
		if len(b) != count*size {
			return nil, fmt.Errorf("chunk tensor size mismatch")
		}
		return b, nil
	}
	floats := func(t tensor) ([]float32, error) {
		b, err := load(t, 4)
		if err != nil {
			return nil, err
		}
		v := make([]float32, len(b)/4)
		for i := range v {
			v[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
		}
		return v, nil
	}
	count := 0
	var tail []float32
	var schedule event
	var segments []dsp.SpeechSegment
	segmentIndex, chunkIndex := 0, 0
	var pcm []byte
	for _, e := range events {
		if e.Event == "synthesis_schedule" {
			if len(segments) > 0 && segmentIndex != len(segments) {
				return fmt.Errorf("incomplete prior schedule")
			}
			schedule = e
			continue
		}
		if e.Event == "torch_jit_result" && len(e.Output.Members) == 7 {
			duration, err := floats(e.Output.Members[0])
			if err != nil {
				return err
			}
			requested := make([]int, len(schedule.Phones))
			for i, p := range schedule.Phones {
				requested[i] = p.Requested
			}
			if len(e.Output.Members[3].Shape) != 3 {
				return fmt.Errorf("invalid acoustic frame shape")
			}
			segments, err = dsp.ScheduleSpeech(duration, schedule.Boundaries, requested, e.Output.Members[3].Shape[2])
			if err != nil {
				return err
			}
			segmentIndex, chunkIndex = 0, 0
			continue
		}
		if e.Event != "chunk_join_parameters" {
			continue
		}
		wave, err := floats(e.Wave)
		if err != nil {
			return err
		}
		previous, err := floats(e.Previous)
		if err != nil {
			return err
		}
		if !e.First {
			if len(tail) != len(previous) {
				return fmt.Errorf("run %d: tail length mismatch", e.Run)
			}
			for i := range tail {
				if tail[i] != previous[i] {
					return fmt.Errorf("run %d: tail differs at sample %d", e.Run, i)
				}
			}
		}
		prefix, err := load(e.Prefix, 2)
		if err != nil {
			return err
		}
		expected, err := load(outputs[e.Run], 2)
		if err != nil {
			return err
		}
		joined, err := dsp.JoinChunk(wave, previous, e.ChunkOptions)
		if err != nil {
			return fmt.Errorf("run %d: %w", e.Run, err)
		}
		tail = joined.Tail
		actual := append([]byte(nil), prefix...)
		for _, v := range joined.PCM {
			actual = binary.LittleEndian.AppendUint16(actual, uint16(v))
		}
		if len(actual) != len(expected) {
			return fmt.Errorf("run %d: got %d samples, native %d; lag %d/%d", e.Run, len(actual)/2, len(expected)/2, joined.CandidateLag, joined.AppliedLag)
		}
		mismatches, maxError := 0, 0
		for i := 0; i < len(actual); i += 2 {
			a := int(int16(binary.LittleEndian.Uint16(actual[i:])))
			b := int(int16(binary.LittleEndian.Uint16(expected[i:])))
			d := a - b
			if d < 0 {
				d = -d
			}
			maxError = max(maxError, d)
			if d != 0 {
				mismatches++
			}
		}
		if maxError != 0 {
			return fmt.Errorf("run %d: %d PCM mismatches, max error %d; lag %d/%d", e.Run, mismatches, maxError, joined.CandidateLag, joined.AppliedLag)
		}
		if segmentIndex >= len(segments) || chunkIndex >= len(segments[segmentIndex].Chunks) {
			return fmt.Errorf("missing derived chunk schedule")
		}
		segment := segments[segmentIndex]
		plan := segment.Chunks[chunkIndex]
		if plan.Start != e.FrameBegin || (plan.End-plan.Start)*240 != len(wave) || plan.Options.EndTrim != e.EndTrim ||
			plan.Options.StartSkip != e.StartSkip || plan.Options.First != e.First || plan.Options.Last != e.Last {
			return fmt.Errorf("run %d: Go chunk plan differs from native: %+v vs %+v", e.Run, plan, e.ChunkOptions)
		}
		if chunkIndex == 0 {
			pcm = append(pcm, make([]byte, segment.PrefixSilence*2)...)
		}
		for _, v := range joined.PCM {
			pcm = binary.LittleEndian.AppendUint16(pcm, uint16(v))
		}
		chunkIndex++
		if chunkIndex == len(segment.Chunks) {
			pcm = append(pcm, make([]byte, segment.SuffixSilence*2)...)
			segmentIndex++
			chunkIndex = 0
		}
		count++
		json.NewEncoder(os.Stdout).Encode(map[string]any{"run": e.Run, "samples": len(actual) / 2, "lag": joined.CandidateLag, "applied_lag": joined.AppliedLag, "close": true})
	}
	if count == 0 {
		return fmt.Errorf("no observed chunk joins")
	}
	if segmentIndex != len(segments) {
		return fmt.Errorf("incomplete derived schedule")
	}
	native, err := os.ReadFile(filepath.Join(root, "probe.wav"))
	if err != nil {
		return err
	}
	wav, err := engine.WAV(pcm, 48000)
	if err != nil {
		return err
	}
	if !bytes.Equal(wav, native) {
		if len(wav) != len(native) {
			return fmt.Errorf("complete WAV length differs: %d vs %d samples", (len(wav)-44)/2, (len(native)-44)/2)
		}
		mismatches, maxError := 0, 0
		for i := 44; i < len(wav); i += 2 {
			d := int(int16(binary.LittleEndian.Uint16(wav[i:]))) - int(int16(binary.LittleEndian.Uint16(native[i:])))
			if d < 0 {
				d = -d
			}
			maxError = max(maxError, d)
			if d > 0 {
				mismatches++
			}
		}
		return fmt.Errorf("complete WAV differs: %d samples, max PCM error %d", mismatches, maxError)
	}
	if err := os.WriteFile(filepath.Join(root, "go-joined.wav"), wav, 0600); err != nil {
		return err
	}
	json.NewEncoder(os.Stdout).Encode(map[string]any{"wave_samples": len(pcm) / 2, "wav_equal": true})
	return nil
}
