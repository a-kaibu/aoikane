package portable

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/a-kaibu/aoikane/internal/engine"
)

// Only the scheduled workflow supplies a corpus. Default unit tests remain
// network-free. Ruby is an independent reading candidate, not proof that an
// isolated word has the same pronunciation in every sentence or edition.
func TestAozoraCorpus(t *testing.T) {
	path := os.Getenv("AOIKANE_AOZORA_CORPUS")
	if path == "" {
		t.Skip("scheduled Aozora corpus not supplied")
	}
	var corpus struct {
		Seed  string
		Books []struct {
			ID, Title, URL string
			Samples        []string
			Rubies         []struct{ Base, Reading, Context string }
		}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &corpus); err != nil {
		t.Fatal(err)
	}
	e, err := Open(Options{AssetsDir: "../../assets"})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	type finding struct{ BookID, URL, Text, Expected, Actual, Context, Status, Voice string }
	report := struct {
		Seed                                                                      string
		Conversions, ConversionFailures, RubyMatches, RubyMismatches, RubySkipped int
		Findings                                                                  []finding
	}{Seed: corpus.Seed}
	defer func() {
		data, _ := json.MarshalIndent(report, "", "  ")
		if output := os.Getenv("AOIKANE_AOZORA_REPORT"); output != "" {
			if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
				t.Error(err)
			} else if err := os.WriteFile(output, data, 0644); err != nil {
				t.Error(err)
			}
		}
		t.Logf("seed=%s conversions=%d failures=%d ruby_match=%d ruby_mismatch=%d ruby_skipped=%d", report.Seed, report.Conversions, report.ConversionFailures, report.RubyMatches, report.RubyMismatches, report.RubySkipped)
	}()
	phones := func(readings []string) ([]int, error) {
		var out []int
		for _, k := range readings {
			if emptyReading(k) {
				continue
			}
			s, err := e.encoder.ParseFormattedKana(k)
			if err != nil {
				return nil, err
			}
			for _, item := range s.Timeline {
				for _, index := range item.PhoneIndices {
					id := e.encoder.Phones[index].ID
					if len(out) > 0 && (id == 3 && out[len(out)-1] == 5 || id == 2 && out[len(out)-1] == 4) {
						id = out[len(out)-1]
					}
					out = append(out, id)
				}
			}
		}
		return out, nil
	}
	for _, book := range corpus.Books {
		for _, voice := range []string{VoiceAoi, VoiceAkane} {
			for i, text := range book.Samples {
				report.Conversions++
				kana, err := e.ReadingVoice(text, voice)
				if err == nil {
					for _, k := range kana {
						if emptyReading(k) {
							continue
						}
						var sErr error
						s, sErr := e.encoder.ParseFormattedKana(k)
						if sErr != nil {
							err = sErr
							break
						}
						if _, sErr = e.encoder.CompilePhones(s, 0xef); sErr != nil {
							err = sErr
							break
						}
					}
				}
				if err == nil {
					p, pErr := phones(kana)
					if pErr != nil {
						err = pErr
					} else if len(p) == 0 {
						err = fmt.Errorf("nonempty sample produced no phonemes")
					}
				}
				if err == nil && i == 0 && os.Getenv("AOIKANE_AOZORA_SYNTH") == "1" {
					_, err = e.Synthesize(context.Background(), engine.Request{Voice: voice, Text: text})
				}
				if err != nil {
					report.ConversionFailures++
					report.Findings = append(report.Findings, finding{BookID: book.ID, URL: book.URL, Text: text, Status: "conversion_error", Actual: err.Error(), Voice: voice})
					t.Errorf("book=%s voice=%s conversion failed: %v", book.ID, voice, err)
				}
			}
		}
		for _, ruby := range book.Rubies {
			reading := strings.Map(func(r rune) rune {
				if r >= 'ぁ' && r <= 'ゖ' {
					return r + 0x60
				}
				return r
			}, ruby.Reading)
			want, werr := phones([]string{"<S>^" + reading + "<N>"})
			actual, aerr := e.ReadingVoice(ruby.Base, VoiceAoi)
			got, gerr := phones(actual)
			status := "match"
			if werr != nil || aerr != nil || gerr != nil || len(got) == 0 {
				report.RubySkipped++
				status = "uncomparable"
			} else if !reflect.DeepEqual(got, want) {
				report.RubyMismatches++
				status = "ruby_difference"
			} else {
				report.RubyMatches++
			}
			if status != "match" {
				report.Findings = append(report.Findings, finding{BookID: book.ID, URL: book.URL, Text: ruby.Base, Expected: ruby.Reading, Actual: strings.Join(actual, " "), Context: ruby.Context, Status: status, Voice: VoiceAoi})
			}
		}
	}
	if report.Conversions == 0 {
		t.Error("corpus contained no conversion samples")
	}
}
