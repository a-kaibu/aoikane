package phrase

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func openPredictor(t *testing.T) *BoundModel {
	t.Helper()
	// A missing library must work: the whole predictor now runs in Go.
	library := filepath.Join(t.TempDir(), "no-runtime-library")
	if _, err := os.Stat(filepath.Join("..", "..", "assets", "boundmodel", "vocab.json")); err != nil {
		t.Skip("decoded predictor models are not available")
	}
	model, err := OpenBoundModel(library, filepath.Join("..", "..", "assets", "boundmodel"), testCodec(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(model.Close)
	return model
}

// TestPredictorMatchesNative runs the rules and the neural predictor on the
// graph the engine had before its BOUND stage and compares the boundary kinds
// with the engine's result.
func TestPredictorMatchesNative(t *testing.T) {
	codec := testCodec(t)
	private := filepath.Join("..", "..", "private")
	library := filepath.Join(private, "runtime", "onnxruntime.dll")
	if _, err := os.Stat(filepath.Join("..", "..", "assets", "boundmodel", "vocab.json")); err != nil {
		t.Skip("decoded predictor models are not available")
	}
	model, err := OpenBoundModel(library, filepath.Join("..", "..", "assets", "boundmodel"), codec)
	if err != nil {
		t.Fatal(err)
	}
	defer model.Close()
	rules := openBoundRules(t, "bound-rules-standard.txt")
	files, _ := filepath.Glob(filepath.Join(private, "stages_aoi", "c*.jsonl"))
	sort.Strings(files)
	total, bad := 0, 0
	for _, path := range files {
		in := snapshotByEvent(t, path, "stage_enter_1b9f60")
		out := snapshotByEvent(t, path, "stage_leave_1b9f60")
		if in == nil || out == nil {
			continue
		}
		doc, err := in.Load(codec)
		if err != nil {
			t.Fatal(err)
		}
		want, err := out.Load(codec)
		if err != nil {
			t.Fatal(err)
		}
		if err := doc.ApplyBound(rules); err != nil {
			t.Fatal(err)
		}
		if err := model.Predict(doc); err != nil {
			t.Fatal(err)
		}
		for i, m := range doc.Morphs {
			w := want.Morphs[i]
			total++
			if m.Kind != w.Kind || m.Kind2 != w.Kind2 {
				bad++
				if bad <= 15 {
					t.Errorf("%s: %q got kind %d/%d want %d/%d", filepath.Base(path), m.Surface, m.Kind, m.Kind2, w.Kind, w.Kind2)
				}
			}
		}
	}
	t.Logf("%d morphs, %d mismatches", total, bad)
}
