package portable

import (
	"bytes"
	"encoding/gob"
	"os"
	"reflect"
	"testing"

	"github.com/a-kaibu/aoikane/internal/frontend"
)

// Cached rule types must survive a gob round trip unchanged.
func TestParsedCacheRoundTrip(t *testing.T) {
	for _, name := range []string{"reading-rules-standard.txt", "reading-rules-kansai.txt"} {
		f, err := os.Open("../../assets/rules/" + name)
		if err != nil {
			t.Fatal(err)
		}
		r, err := frontend.ParseReadingRules(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		if err := gob.NewEncoder(&buf).Encode(r); err != nil {
			t.Fatal(err)
		}
		var out frontend.ReadingRules
		if err := gob.NewDecoder(&buf).Decode(&out); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(*r, out) {
			t.Fatalf("%s differs after a gob round trip", name)
		}
	}
}
