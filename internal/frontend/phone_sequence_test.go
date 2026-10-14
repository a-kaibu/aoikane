package frontend

import (
	"slices"
	"testing"
)

func TestNewlineEndNativePhones(t *testing.T) {
	e, err := OpenMoraEncoder("../../assets/data/phonetics.json")
	if err != nil {
		t.Fatal(err)
	}
	s, err := e.ParseFormattedKana("<S>^ゼ!ロ<N>")
	if err != nil {
		t.Fatal(err)
	}
	got, err := e.CompilePhones(s, 0xef)
	if err != nil {
		t.Fatal(err)
	}
	// Normal compileToImphoneme and model inputs for seed19 case58, 2026-10-13.
	// Observation's complete WAV also matches the independently captured SDK WAV.
	if !slices.Equal(got.Phoneme, []int64{0, 15, 4, 45, 5, 0}) ||
		!slices.Equal(got.Accent, []int64{0, 3, 3, 1, 1, 0}) ||
		!slices.Equal(got.Punctuation, []int64{0, 1, 1, 1, 1, 2}) ||
		!slices.Equal(got.MoraPosition, []float32{0, .25, .25, .75, .75, 0}) ||
		!slices.Equal(got.PhrasePosition, []float32{0, .5, .5, .5, .5, 0}) {
		t.Fatalf("newline end model tokens differ from original: %+v", got.ProsodyTokens)
	}
	last := len(got.Phones) - 1
	want := Allophone{ID: 0, Label: "--_N", Info: 0xcf00, BoundaryCode: 0x15}
	if got.Phones[last] != want || got.Articulation[last] != 0x40000001 {
		t.Fatalf("newline end: got %+v art=%08x, want %+v art=40000001", got.Phones[last], got.Articulation[last], want)
	}
}

func TestFricativeGeminateNativePhone(t *testing.T) {
	e, err := OpenMoraEncoder("../../assets/data/phonetics.json")
	if err != nil {
		t.Fatal(err)
	}
	s, err := e.ParseFormattedKana("<S>フ^ロルプト|0ス^ロ!ッシュト|0ス^ナ!ールフオ|0ニュ^ーリョクシタ<F>")
	if err != nil {
		t.Fatal(err)
	}
	got, err := e.CompilePhones(s, 0xef)
	if err != nil {
		t.Fatal(err)
	}
	// Original compileToImphoneme capture, 2026-10-08. The mora-head
	// flag is packed separately in native JSON and excluded from Info here.
	want := Allophone{ID: 14, Label: "Qsh_", Info: 0x4340}
	if got.Phones[15] != want || got.Articulation[15] != 0x0c402014 {
		t.Fatalf("geminate: got %+v art=%08x, native %+v art=0c402014", got.Phones[15], got.Articulation[15], want)
	}
}

func TestPauseVoicingBesideGeminateMatchesNative(t *testing.T) {
	e, err := OpenMoraEncoder("../../assets/data/phonetics.json")
	if err != nil {
		t.Fatal(err)
	}
	s, err := e.ParseFormattedKana("<S>^ア!ッア|0_2^ア!ッン<F>")
	if err != nil {
		t.Fatal(err)
	}
	got, err := e.CompilePhones(s, 0xef)
	if err != nil {
		t.Fatal(err)
	}
	// Same pause fields as the original 2026-10-08 geminate/vowel trace.
	want := Allophone{ID: 0, Label: "--_-", Info: 0x8f00, BoundaryCode: 0x14}
	if got.Phones[4] != want || got.Articulation[4] != 0x80000009 {
		t.Fatalf("pause: got %+v art=%08x, native %+v art=80000009", got.Phones[4], got.Articulation[4], want)
	}
}
