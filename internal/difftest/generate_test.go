package difftest

import (
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestReproducibleCorpus(t *testing.T) {
	a := Texts(1, 128, 6)
	if !reflect.DeepEqual(a, Texts(1, 128, 6)) {
		t.Fatal("same seed changed corpus")
	}
	if reflect.DeepEqual(a, Texts(2, 128, 6)) {
		t.Fatal("different seed did not change corpus")
	}
	if !reflect.DeepEqual(a[:8], Texts(1, 8, 6)) {
		t.Fatal("case indices changed when count changed")
	}
	for _, text := range a {
		if !utf8.ValidString(text) || !strings.HasSuffix(text, "。") {
			t.Fatalf("invalid generated input %q", text)
		}
	}
	// Ensure the raw-Unicode mode actually reaches characters outside CP932,
	// rather than merely testing templates or kana.
	found := false
	for i, text := range Texts(42, 1000, 8) {
		if i%4 == 2 && strings.ContainsRune(text, '😀') {
			found = true
		}
	}
	if !found {
		t.Fatal("Unicode stress characters absent")
	}
}
