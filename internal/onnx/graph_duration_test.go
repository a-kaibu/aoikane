package onnx

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"slices"
	"testing"
)

func TestDurationGraphMatchesOriginal(t *testing.T) {
	testDurationAndFrameGraph(t, 102, false)
}

func TestF0PrefixGraphMatchesOriginal(t *testing.T) {
	testDurationAndFrameGraph(t, 176, false)
}

func TestFrameBoundariesMatchOriginal(t *testing.T) {
	testDurationAndFrameGraph(t, 190, true)
}

func TestF0ExpansionMatchesOriginal(t *testing.T) {
	testDurationAndFrameGraph(t, 190, false)
}

func TestF0FrameStackMatchesOriginal(t *testing.T) {
	testDurationAndFrameGraph(t, 214, false)
}

func TestPhonemeF0MatchesOriginal(t *testing.T) {
	testDurationAndFrameGraph(t, 233, false)
}

func TestFrameF0MatchesOriginal(t *testing.T) {
	testDurationAndFrameGraph(t, 398, false)
}

func TestCompleteFeatNetMatchesOriginal(t *testing.T) {
	testDurationAndFrameGraph(t, 662, false)
}

func testDurationAndFrameGraph(t *testing.T, last int, boundaries bool) {
	t.Helper()
	g, err := OpenGraph("../../assets/synth/featnet.onnx")
	if err != nil {
		t.Fatal(err)
	}
	var outputs []string
	for i, n := range g.nodes {
		if i >= 10 && i <= 49 && n.op != "Constant" {
			outputs = append(outputs, n.outputs[0])
		}
	}
	outputs = append(outputs, "/model/Unsqueeze_output_0")
	for i, n := range g.nodes {
		if i >= 69 && i <= last && n.op != "Constant" && i != 101 {
			outputs = append(outputs, n.outputs[0])
		}
	}
	outputs = append(outputs, "/model/dec_dur/Greater_output_0")
	if last == 662 {
		outputs = []string{"duration", "phoneme_f0", "phoneme_power", "f0", "power", "bap", "spc"}
	}
	wantOutputs := map[int]int{102: 64, 176: 119, 190: 127, 214: 151, 233: 165, 398: 261, 662: 7}[last]
	if len(outputs) != wantOutputs {
		t.Fatalf("duration output count %d", len(outputs))
	}
	hash, final := sha256.New(), sha256.New()
	count := 0
	lengths := []int{1, 2, 5, 17, 65, 257}
	if last == 662 {
		// The complete decoder is costly. Longer cases are covered by the
		// explicit native comparison tool, without making every unit run slow.
		lengths = []int{2, 5, 17}
	}
	for _, length := range lengths {
		// synthesis.Input.tensors already rejects fewer than two phonemes.
		// At length one the original frame-F0 Pad crops the whole input and
		// leaves output storage unwritten. Keep the raw length-one corpus in
		// earlier stages, without pretending it has a defined frame-F0 oracle.
		if last >= 398 && length == 1 {
			continue
		}
		for variant := 0; variant < 3; variant++ {
			in := embeddingInputs(length)
			for _, name := range []string{"speed", "pitch", "into", "loud"} {
				v := Value{Kind: 1, Shape: []int64{1, 1, int64(length)}, F: make([]float32, length)}
				for i := range v.F {
					v.F[i] = 1
				}
				in[name] = v
			}
			sty := Value{Kind: 1, Shape: []int64{1, 8, int64(length)}, F: make([]float32, 8*length)}
			dur := Value{Kind: 1, Shape: []int64{1, 1, int64(length)}, F: make([]float32, length)}
			for i := range dur.F {
				dur.F[i] = -1
			}
			if variant > 0 {
				for i := range sty.F {
					sty.F[i] = float32(i%17-8) * float32(variant) / 16
				}
				for i := range dur.F {
					switch i % 4 {
					case 0:
						dur.F[i] = -1
					case 1:
						dur.F[i] = 0
					case 2:
						dur.F[i] = float32(i%7+1) / 100
					case 3:
						dur.F[i] = 0.5
					}
				}
			}
			if variant == 2 {
				for _, name := range []string{"speed", "pitch", "into", "loud"} {
					for i := range in[name].F {
						in[name].F[i] = float32(i%7+1) / 4
					}
				}
				for i := range in["phn"].I {
					in["phn"].I[i] = int64((i*17 + 3) % 52)
					in["acc"].I[i] = int64((i*3 + 1) % 4)
					in["punc"].I[i] = int64((i*5 + 1) % 8)
				}
				for i := range in["art"].F {
					in["art"].F[i] = float32(i%17-8) / 8
				}
			}
			in["sty"], in["dur"] = sty, dur
			if last >= 233 {
				v := Value{Kind: 1, Shape: []int64{1, 1, int64(length)}, F: make([]float32, length)}
				for i := range v.F {
					v.F[i] = -1
				}
				if variant != 0 {
					overrides := []float32{-1, 0, 19.999, 20, 20.001, 100, 220, 440, 880, 3000}
					for i := range v.F {
						v.F[i] = overrides[(i+variant)%len(overrides)]
					}
				}
				in["f0c"] = v
			}
			if last == 662 {
				v := Value{Kind: 1, Shape: []int64{1, 1, int64(length)}, F: make([]float32, length)}
				for i := range v.F {
					v.F[i] = -1
					if variant != 0 {
						overrides := []float32{-1, -0.001, 0, 0.001, 0.1, 0.5, 1, 2, 10}
						v.F[i] = overrides[(i+variant)%len(overrides)]
					}
				}
				in["pwr"] = v
			}
			if boundaries {
				values := []float32{2.5, 5, 7.5, 12.5, 0.0001, 100}
				for i := range dur.F {
					dur.F[i] = values[(i+variant)%len(values)]
				}
			}
			out, err := g.Run(in, outputs)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(out["duration"].Shape, []int64{1, 1, int64(length)}) {
				t.Fatal("duration shape differs")
			}
			for i, x := range dur.F {
				if x > 0 && math.Float32bits(out["duration"].F[i]) != math.Float32bits(x) {
					t.Fatal("duration override differs")
				}
			}
			for _, name := range outputs {
				v := out[name]
				values := v.F
				if v.Kind == 9 {
					values = make([]float32, len(v.B))
					for i, b := range v.B {
						if b {
							values[i] = 1
						}
					}
				}
				if v.Kind == 7 {
					values = make([]float32, len(v.I))
					for i, x := range v.I {
						values[i] = float32(x)
						if int64(values[i]) != x {
							t.Fatal("integer observation loses precision")
						}
					}
				}
				for _, x := range values {
					hash.Write(binary.LittleEndian.AppendUint32(nil, math.Float32bits(x)))
				}
				count += len(values)
			}
			finalValues := out["duration"].F
			if last == 233 {
				finalValues = out["phoneme_f0"].F
				if !slices.Equal(out["phoneme_f0"].Shape, []int64{1, 1, int64(length)}) {
					t.Fatal("phoneme F0 shape differs")
				}
			}
			if last == 398 {
				finalValues = out["f0"].F
				var frames int64
				for _, x := range out["/model/cnv_rep/Cast_1_output_0"].I {
					frames += x
				}
				if !slices.Equal(out["f0"].Shape, []int64{1, 1, frames}) {
					t.Fatal("frame F0 shape differs")
				}
			}
			for _, x := range finalValues {
				final.Write(binary.LittleEndian.AppendUint32(nil, math.Float32bits(x)))
			}
		}
	}
	// Migrated intermediates independently recorded from original Runtime.
	// The duration case also checks the final output from
	// the unmodified complete original model.
	want, wantCount := "82a29006138aa0ea0bdbf208d4eb1e2d91f89c5c1efff35c41c5a8e56e8241c2", 6709245
	if last == 176 {
		want, wantCount = "f003d522b408605490af60e9c1b4c19e3004c284cd8eb07eadff6e6f29e9bc1d", 9260433
	}
	if last == 190 {
		want, wantCount = "8c2f3d631f3c41d6688aff9617fad53f9d8117e2de335783cb53bfaa72215c88", 27135316
	}
	if last == 214 {
		want, wantCount = "80642d1f622fbd877cb4a319f8f4b8416b9948fc8da3210a21da7ff59efcc28d", 56385840
	}
	if last == 233 {
		want, wantCount = "a32b7c1338b4f5a0a70f0ec8acef12472719ec294c7439ef636fcb5ce49f28d5", 56400414
	}
	if last == 398 {
		want, wantCount = "50dbed319d65c17b5eec24c39281ef651bcba6346b9e14ffb818d92d2956b9af", 118053815
	}
	if last == 662 {
		want, wantCount = "c677d9bcb7bef3ffcc0c141e1e12b424a3f1cc2a435c089caa4906c772027a6a", 245369
	}
	if boundaries {
		want, wantCount = "ca5a8ad3d08eb389d2cc3908e331f4580da60d59ee43c096d5293a6d151cf1e3", 11348714
	}
	if got := fmt.Sprintf("%x", hash.Sum(nil)); got != want || count != wantCount {
		t.Fatalf("%d duration values hash %s want %s", count, got, want)
	}
	finalWant := "3dfa5c34112d8421fbab00198294c6ee9c1ccedbc836dd6c6a58266050a37786"
	if last == 233 {
		finalWant = "0f7fb2f77dd3369a84c8a72735963ebf33c141ca12bff7459a3f5f989391a36c"
	}
	if last == 398 {
		finalWant = "054cab019c60d063b0a5adf2956dc584c2f455b997390d55aabc8eba2b8e3f79"
	}
	if got := fmt.Sprintf("%x", final.Sum(nil)); (last == 102 || last == 233 || last == 398) && got != finalWant {
		t.Fatalf("original duration SHA256 %s want %s", got, finalWant)
	}
}
