package frontend

import "testing"

func TestLongMoraNativePhones(t *testing.T) {
	e, err := OpenMoraEncoder("../../assets/data/phonetics.json")
	if err != nil {
		t.Fatal(err)
	}
	// Fields from independent native compileToImphoneme captures of the
	// corresponding text, including the final post-allophone closure pass.
	for _, tc := range []struct {
		kana         string
		count, index int
		phone        Allophone
		art          uint32
		position     float32
	}{
		{"<S>^ー!カナデス<F>", 10, 0, Allophone{ID: 0, Label: "--_*", Info: 0xcf00, BoundaryCode: 12}, 0x20000001, 0},
		{"<S>^ン!ーカナデス<F>", 12, 1, Allophone{ID: 50, Label: "Ng:*", Info: 0x1430, BoundaryCode: 12}, 0x0111000c, float32(0.5) / 6},
		{"<S>^ッ!ーカナデス<F>", 12, 1, Allophone{ID: 51, Label: "Q__*", Info: 0x0f50, BoundaryCode: 12}, 0x04000004, float32(0.5) / 6},
		{"<S>^ン!ンカナデス<F>", 12, 1, Allophone{ID: 50, Label: "Ng_*", Info: 0x1420, BoundaryCode: 12}, 0x0011000c, float32(0.5) / 6},
	} {
		t.Run(tc.kana, func(t *testing.T) {
			s, err := e.ParseFormattedKana(tc.kana)
			if err != nil {
				t.Fatal(err)
			}
			got, err := e.CompilePhones(s, 0xef)
			if err != nil {
				t.Fatal(err)
			}
			if len(got.Phones) != tc.count || got.Phones[tc.index] != tc.phone || got.Articulation[tc.index] != tc.art || got.MoraPosition[tc.index] != tc.position {
				t.Fatalf("got %d phones, %+v art=%08x position=%g; native %d, %+v art=%08x position=%g", len(got.Phones), got.Phones[tc.index], got.Articulation[tc.index], got.MoraPosition[tc.index], tc.count, tc.phone, tc.art, tc.position)
			}
		})
	}
}
