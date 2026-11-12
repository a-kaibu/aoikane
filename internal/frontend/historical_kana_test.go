package frontend

import (
	"reflect"
	"testing"
)

func TestFormattedKanaHistoricalVowels(t *testing.T) {
	e, err := OpenMoraEncoder("../../assets/data/phonetics.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]string{{"<S>^ヲヰヱ<N>", "<S>^オイエ<N>"}, {"<S>ジョ^ーブナカラダヲ!モチ<N>", "<S>ジョ^ーブナカラダオ!モチ<N>"}, {"<S>イ^ツモシヅカニワラッテ!ヰル<N>", "<S>イ^ツモシヅカニワラッテ!イル<N>"}} {
		got, err := e.ParseFormattedKana(pair[0])
		if err != nil {
			t.Fatal(err)
		}
		want, err := e.ParseFormattedKana(pair[1])
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("historical vowel changed accent/mora structure: %s", pair[0])
		}
		p, err := e.CompilePhones(got, 0xef)
		if err != nil {
			t.Fatal(err)
		}
		q, err := e.CompilePhones(want, 0xef)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(p, q) {
			t.Fatal("historical pronunciation differs from modern vowels")
		}
	}
}
