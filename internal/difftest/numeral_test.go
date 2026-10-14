package difftest

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

func TestNumeralCorpusReplay(t *testing.T) {
	texts := NumeralTexts(1, 128, 16)
	raw, err := json.Marshal(texts)
	if err != nil {
		t.Fatal(err)
	}
	// A pinned stream ensures a saved failing index keeps the same full input.
	if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != "dfea907dbd7c8230ef85710c524c791328af59d134c7b48fc350759aeef98ffd" {
		t.Fatalf("numeral generator version 1 changed: %s", got)
	}
	if !reflect.DeepEqual(texts[:8], NumeralTexts(1, 8, 16)) {
		t.Fatal("changing count changes replay indices")
	}
	if reflect.DeepEqual(texts, NumeralTexts(2, 128, 16)) {
		t.Fatal("different seed did not change inputs")
	}
}
