package portable

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestNewlineReadingOriginal(t *testing.T) {
	raw, err := os.ReadFile("testdata/newline_reading_original.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		DLLHash  string `json:"dll_sha256"`
		Voice    string
		Captures []string `json:"capture_sha256"`
		Cases    []struct {
			Text     string
			Readings []string
		}
	}
	if err = json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.DLLHash != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || ref.Voice != VoiceAkane || len(ref.Captures) != 4 || len(ref.Cases) != 21 {
		t.Fatal("original newline reference provenance differs")
	}
	e, err := Open(Options{AssetsDir: "../../assets"})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	for _, c := range ref.Cases {
		got, err := e.ReadingVoice(c.Text, ref.Voice)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, c.Readings) {
			t.Errorf("%q: got %q want %q", c.Text, got, c.Readings)
		}
	}
}
