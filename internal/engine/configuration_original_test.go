package engine

import (
	"encoding/json"
	"math"
	"os"
	"strconv"
	"testing"
)

type configReferenceCase struct {
	Parameter string `json:"parameter"`
	Bits      string `json:"value_bits"`
	Status    int32  `json:"status"`
}

func originalConfigReference(t *testing.T) []configReferenceCase {
	t.Helper()
	raw, err := os.ReadFile("testdata/original_config_bounds.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		DLLHash    string `json:"dll_sha256"`
		SourceHash string `json:"source_sha256"`
		Cases      []configReferenceCase
	}
	if err = json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.DLLHash != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(ref.SourceHash) != 64 || len(ref.Cases) != 96 {
		t.Fatal("original setter provenance differs")
	}
	return ref.Cases
}
func TestOriginalConfigurationValidation(t *testing.T) {
	checked := 0
	for _, c := range originalConfigReference(t) {
		bits, err := strconv.ParseUint(c.Bits, 16, 64)
		if err != nil {
			t.Fatal(err)
		}
		v := math.Float64frombits(bits)
		// Raw SDK accepts NaN; this API has always rejected nonfinite controls.
		// Zero requests are defaults, not raw setter-zero calls.
		if math.IsNaN(v) || v == 0 {
			continue
		}
		r := Request{Text: "あ。", Rate: 1, Pitch: 1, Volume: 1, Emphasis: 1}
		switch c.Parameter {
		case "Rate":
			r.Rate = v
		case "Pitch":
			r.Pitch = v
		case "Volume":
			r.Volume = v
		case "Emph":
			r.Emphasis = v
		default:
			t.Fatal("unknown original control")
		}
		if accepted := r.Validate() == nil; accepted != (c.Status == 0) {
			t.Errorf("%s value=%g: Go accepted=%v, native status=%d", c.Parameter, v, accepted, c.Status)
		}
		checked++
	}
	if checked != 88 {
		t.Fatalf("checked %d original values", checked)
	}
}
func TestConfigurationZeroDefaults(t *testing.T) {
	r := Request{Text: "あ。"}
	r.Defaults()
	if r.Rate != 1 || r.Pitch != 1 || r.Volume != 1 || r.Emphasis != 1 || r.Validate() != nil {
		t.Fatal("zero-as-default request contract differs")
	}
}
