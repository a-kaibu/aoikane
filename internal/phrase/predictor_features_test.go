package phrase

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"math"
	"os"
	"testing"
)

// Compare the actual model inputs, including the detached numeral chain's
// mora count, against a fresh observation of the native boundary predictor.
func TestPhoneBoundFeaturesMatchesNative(t *testing.T) {
	for _, tc := range []struct{ path, text string }{
		{"../../private/phone-bound-fresh.jsonl", "電話は03-5555-1234です。"},
		{"../../private/tollfree-bound-fresh.jsonl", "フリーダイヤルは0120-456-789です。"},
	} {
		t.Run(tc.text, func(t *testing.T) { compareBoundFeatures(t, tc.path, tc.text) })
	}
}

func compareBoundFeatures(t *testing.T, path, text string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skip("fresh native boundary features are not available")
	}
	var native struct {
		Event, IDs, Floats string
	}
	if err := json.Unmarshal(raw, &native); err != nil {
		t.Fatal(err)
	}
	if native.Event != "features" {
		t.Fatalf("capture event %q, want features", native.Event)
	}
	fe := openFrontend(t)
	model := openPredictor(t)
	doc, err := fe.Graph(text)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.ApplyBound(openBoundRules(t, "bound-rules-standard.txt")); err != nil {
		t.Fatal(err)
	}
	words := doc.Morphs[:len(doc.Morphs)-1]
	ids, features, err := model.features(words)
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(i64bytes(ids)); got != native.IDs {
		t.Fatalf("token ids: got %s; native %s", got, native.IDs)
	}
	out, err := model.emb.Lookup(ids)
	if err != nil {
		t.Fatal(err)
	}
	for i := range words {
		copy(features[i*featureDims:], out[i*embedDims:(i+1)*embedDims])
	}
	want, err := hex.DecodeString(native.Floats)
	if err != nil {
		t.Fatal(err)
	}
	if len(want) != len(features)*4 {
		t.Fatalf("native bytes %d, generated features %d", len(want), len(features))
	}
	bad := 0
	for i, got := range features {
		bits := binary.LittleEndian.Uint32(want[4*i:])
		if math.Float32bits(got) != bits {
			bad++
			t.Errorf("%q feature %d: got %.9g (%08x); native %.9g (%08x)", words[i/featureDims].Surface, i%featureDims, got, math.Float32bits(got), math.Float32frombits(bits), bits)
		}
	}
	t.Logf("%d token ids and %d features compared; %d differences", len(ids), len(features), bad)
}

func TestMoraFeatureExpansionTail(t *testing.T) {
	for _, tc := range []struct {
		count int
		want  int32
	}{
		{0, 2}, {1, 6}, {2, 6}, {3, 10},
	} {
		m := &Morph{Moras: 2}
		var last *Morph
		for i := 0; i < tc.count; i++ {
			x := &Morph{Cands: []Candidate{{Pronunciation: "サンヨン"}}}
			if last == nil {
				m.ExpFirst = x
			} else {
				last.Next = x
			}
			last = x
		}
		m.ExpLast = last
		// Each pronunciation has four moras. A single node is visited, while
		// the tail of a longer detached chain is excluded by native 18fa60.
		if got := moraFeature(m); got != tc.want {
			t.Errorf("chain length %d: got %d; native %d", tc.count, got, tc.want)
		}
	}
}
