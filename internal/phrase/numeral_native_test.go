package phrase

import (
	"os"
	"testing"

	"github.com/a-kaibu/aoikane/internal/numeral"
)

// Candidate values observed in fresh BANCHA output from the native engine.
// Keep these compact expectations runnable without private trace files.
func TestEnglishNumeralCandidate(t *testing.T) {
	f, err := os.Open("../../assets/rules/bancharcx-standard.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rules, err := numeral.ParseRules(f)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		number string
		want   Candidate
	}{
		{"３", Candidate{"３スリー", "スリー", "2-2-3", "F4@2"}},
		{"５", Candidate{"ファイブ", "ファイブ", "1-1-3", "*"}},
	} {
		t.Run(tc.number, func(t *testing.T) {
			readings, err := rules.ReadEnglish(tc.number)
			if err != nil {
				t.Fatal(err)
			}
			m, chain := numeralMorph(tc.number, readings)
			if len(chain) != 0 || len(m.Cands) != 1 || m.Cands[0] != tc.want {
				t.Fatalf("got %+v, chain length %d; native %+v", m.Cands, len(chain), tc.want)
			}
		})
	}
}

// Fresh snapshots come from the unmodified native engine, one process per
// sentence. Unlike batch captures, their sentence index restarts at zero.
func TestEnglishNumeralMatchesNative(t *testing.T) {
	compareFreshFrontend(t, "../../private/number-prefix-fresh.jsonl")
}

func TestPhoneFrontendMatchesNative(t *testing.T) {
	compareFreshFrontend(t, "../../private/phone-graph-fresh.jsonl")
}

func compareFreshFrontend(t *testing.T, path string) {
	if _, err := os.Stat(path); err != nil {
		t.Skip("fresh native number-prefix capture is not available")
	}
	snaps, err := ReadSnapshots(path)
	if err != nil {
		t.Fatal(err)
	}
	codec := testCodec(t)
	fe := openFrontend(t)
	fe.Bound = openBoundRules(t, "bound-rules-standard.txt")
	fe.Predictor = openPredictor(t)
	fe.Reading.SkipPhraseScopes = true
	var leave *Snapshot
	var bound *Snapshot
	total := 0
	for i := range snaps {
		s := &snaps[i]
		if s.Event == "stage_leave_1b9f10" {
			leave = s
		}
		if s.Event == "stage_leave_1b9f60" {
			bound = s
		}
		if s.Event != "format" {
			continue
		}
		if leave == nil || leave.Input != s.Input {
			t.Fatalf("missing BANCHA snapshot for %q", s.Input)
		}
		t.Run(s.Input, func(t *testing.T) {
			want, err := leave.Load(codec)
			if err != nil {
				t.Fatal(err)
			}
			got, err := fe.Graph(s.Input)
			if err != nil {
				t.Fatal(err)
			}
			if diff := compareGraphs(got, want); diff != "" {
				t.Error(diff)
			}
			if bound != nil {
				wantBound, err := bound.Load(codec)
				if err != nil {
					t.Fatal(err)
				}
				if err := got.ApplyBound(fe.Bound); err != nil {
					t.Fatal(err)
				}
				if err := fe.Predictor.Predict(got); err != nil {
					t.Fatal(err)
				}
				for i, m := range got.Morphs {
					w := wantBound.Morphs[i]
					if m.Kind != w.Kind || m.Kind2 != w.Kind2 {
						t.Errorf("%q boundary got %d/%d; native %d/%d", m.Surface, m.Kind, m.Kind2, w.Kind, w.Kind2)
					}
				}
			}
			wantText, err := s.FormatText(codec)
			if err != nil {
				t.Fatal(err)
			}
			gotText, err := fe.Kana(s.Input)
			if err != nil {
				t.Fatal(err)
			}
			if gotText != wantText {
				t.Errorf("got %s; native %s", gotText, wantText)
			}
			t.Logf("native reading: %s", wantText)
		})
		total++
		leave = nil
		bound = nil
	}
	if total == 0 {
		t.Fatal("capture contains no formatted readings")
	}
	t.Logf("compared %d native graphs and formatted readings", total)
}
