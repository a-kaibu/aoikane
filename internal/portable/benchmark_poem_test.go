package portable

import (
	"os"
	"strings"
	"testing"
)

func TestBenchmarkPoemKanaIsEncodable(t *testing.T) {
	b, err := os.ReadFile("../../benchmarks/texts/ame-ni-mo-makezu.txt")
	if err != nil {
		t.Fatal(err)
	}
	e, err := Open(Options{AssetsDir: "../../assets"})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	for _, voice := range []string{VoiceAoi, VoiceAkane} {
		for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
			kana, err := e.ReadingVoice(line, voice)
			if err != nil {
				t.Fatal(line, "READING", err)
			}
			for _, k := range kana {
				parsed, err := e.encoder.ParseFormattedKana(k)
				if err != nil {
					t.Fatalf("%q %q: %v", line, k, err)
				}
				if _, err := e.encoder.CompilePhones(parsed, 0xef); err != nil {
					t.Fatal(line, err)
				}
			}
		}
	}
}
