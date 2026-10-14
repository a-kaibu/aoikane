package phrase

import (
	"github.com/a-kaibu/aoikane/internal/frontend"
	"encoding/json"
	"os"
	"testing"
)

func TestKansaiMorasOriginal(t *testing.T) {
	raw, err := os.ReadFile("testdata/kansai_mora_counts.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		DLLHash string `json:"dll_sha256"`
		Entry   string `json:"entry_rva"`
		Cases   []struct {
			Text  string
			Count int32
		}
	}
	if err = json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.DLLHash != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || ref.Entry != "0x1962d0" || len(ref.Cases) != 523 {
		t.Fatal("invalid original mora reference")
	}
	codec, err := frontend.OpenCP932Codec("../../assets/data/cp932.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range ref.Cases {
		got, err := kansaiMoras(c.Text, codec)
		if err != nil {
			t.Fatal(err)
		}
		if got != c.Count {
			t.Errorf("%q got %d want %d", c.Text, got, c.Count)
		}
	}
}
