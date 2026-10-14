package portable

import (
	"github.com/a-kaibu/aoikane/internal/engine"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"testing"
)

func TestReiReadingOriginal(t *testing.T) {
	raw, err := os.ReadFile("testdata/rei_reading_original.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		DLL      string   `json:"dll_sha256"`
		Captures []string `json:"capture_sha256"`
		Cases    []struct {
			Text, Voice string
			Readings    []string
		}
	}
	if err = json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.DLL != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(ref.Captures) != 6 || len(ref.Cases) != 136 {
		t.Fatal("original reading capture provenance differs")
	}
	e, err := Open(Options{AssetsDir: "../../assets"})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	for _, c := range ref.Cases {
		got, err := e.ReadingVoice(c.Text, c.Voice)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(got, c.Readings) {
			t.Errorf("%s %q: got %q; want %q", c.Voice, c.Text, got, c.Readings)
		}
	}
}

func TestRandomSeed5ReadingOriginal(t *testing.T) {
	testRandomReadingOriginal(t, 5, "1f6c005ee623780079a71598ef7f879415910a109193c9dfd87cdebd0ad73e55")
}

func TestRandomSeed6ReadingOriginal(t *testing.T) {
	testRandomReadingOriginal(t, 6, "ea6c19cd6e772f7f3097c22128d1617a3361adbef696ad27203f8b01805a0692")
}

func TestRandomSeed7ReadingOriginal(t *testing.T) {
	testRandomReadingOriginal(t, 7, "0f58da44de83bd2a270eb17ed5a69c17e2d64daa8a010f1a20aa4562d2ce8319")
}

func TestRandomSeed8ReadingOriginal(t *testing.T) {
	testRandomReadingOriginal(t, 8, "43e28aecb982c6600591bb1d0d016c7282d67b720c28b547e9d268503be37913")
}

func TestRandomSeed9ReadingOriginal(t *testing.T) {
	testRandomReadingOriginal(t, 9, "b68cb943291d099d50865e186f56c4b691f31786fa5490799005897b13e005d4")
}

func TestRandomSeed10ReadingOriginal(t *testing.T) {
	testRandomReadingOriginal(t, 10, "8012ad089fd2dd4dfd7aee8f1ae7a253e50b090a4a521d68397b82c04405eec4")
}

func TestRandomSeed11ReadingOriginal(t *testing.T) {
	testRandomReadingOriginal(t, 11, "987bbf313c857f4ca2b336dec85b04b6916c142fe52ab459be8457bac7b96bc8")
}

func TestRandomSeed12ReadingOriginal(t *testing.T) {
	testRandomReadingOriginal(t, 12, "e9c190909d43f0d9be17542fd4e4ec8cc8598ecbc9b94953e50e874ab8cae246")
}

func TestRandomSeed13LongReadingOriginal(t *testing.T) {
	testRandomReadingOriginal(t, 13, "1626e664afb974622fa0cc6e71214e6175f4de99c8adf802877e5a6ecc96c8b7")
}

func TestRandomSeed14LongReadingOriginal(t *testing.T) {
	testRandomReadingOriginal(t, 14, "42de873c17a87879dc67ff9bd4adf60b8460fa58578bf687f8761d8d09ee26b6")
}

func TestRandomSeed15LongReadingOriginal(t *testing.T) {
	testRandomReadingOriginal(t, 15, "fc85b1ae1fbbbb6008632bd5b667f0bb5057fca3b37eed655f35dba6eadc6a88")
}

func TestRandomSeed16LongReadingOriginal(t *testing.T) {
	testRandomReadingOriginal(t, 16, "1bdddd32b467d0350e6d1c8fb6dbe8769c354044cd43ec7581ebe84910121ee9")
}

func TestRandomSeed17LongReadingOriginal(t *testing.T) {
	testRandomReadingOriginal(t, 17, "ee73b8e21904f951431a180325fd92c3f0aed8813133d120d0177a4cd42579cc")
}

func TestRandomSeed18LongReadingOriginal(t *testing.T) {
	testRandomReadingOriginal(t, 18, "830945b988184ac818173d53102728382374cd36ecc9118f99604124c71e9f5b")
}

func TestRandomSeed19LongReadingOriginal(t *testing.T) {
	testRandomReadingOriginal(t, 19, "006e062a26565e8a79235bcf9461a6e13f50dc91bd4abbc5598b78a279b10b40")
}

func TestRandomSeed20LongReadingOriginal(t *testing.T) {
	testRandomReadingOriginal(t, 20, "1feafd104c5acb073b6d3dc75d4e660238dfa6b7e3b3bc33a70dbc78dbb846f5")
}

func TestRandomSeed21LongReadingOriginal(t *testing.T) {
	testRandomReadingOriginal(t, 21, "d6df7fb2bec670dcef249048566efd02b72c38b18bc1441af1304c1b692b8286")
}

func TestRandomSeed22LongReadingOriginal(t *testing.T) {
	testRandomReadingOriginal(t, 22, "e57ca1c6ef7a10b52d5dbd40ff2d27fdfb4766e8a6e030046cff51bc933b9500")
}

func TestRandomSeed23LongReadingOriginal(t *testing.T) {
	testRandomReadingOriginal(t, 23, "5d6863ef9b0c8077c509016280fff299a3dcd479ba10e75c2bd661247287126e")
}

func testRandomReadingOriginal(t *testing.T, seed int, corpus string) {
	t.Helper()
	e, err := Open(Options{AssetsDir: "../../assets"})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	for _, voice := range []string{"akane", "aoi"} {
		raw, err := os.ReadFile(fmt.Sprintf("../../tools/compare_reading/testdata/seed%d-%s-original.json", seed, voice))
		if err != nil {
			t.Fatal(err)
		}
		var ref struct {
			Voice    string
			DLL      string `json:"dll_sha256"`
			Corpus   string `json:"corpus_sha256"`
			Complete bool
			Cases    []struct {
				Index    int
				Text     string
				Readings []string
			}
		}
		if err = json.Unmarshal(raw, &ref); err != nil {
			t.Fatal(err)
		}
		wantVoice := VoiceAkane
		if voice == "aoi" {
			wantVoice = VoiceAoi
		}
		if !ref.Complete || ref.Voice != wantVoice || ref.DLL != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || ref.Corpus != corpus || len(ref.Cases) != 128 {
			t.Fatal("original random capture is incomplete")
		}
		for i, c := range ref.Cases {
			if c.Index != i || c.Text == "" || len(c.Readings) == 0 {
				t.Fatal("original random ordering differs")
			}
			got, err := e.ReadingVoice(c.Text, ref.Voice)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, c.Readings) {
				t.Errorf("%s case %d %q: got %q; want %q", ref.Voice, i, c.Text, got, c.Readings)
			}
		}
	}
}

func TestReiOriginalWAV(t *testing.T) {
	if os.Getenv("AOIKANE_REI_REFERENCE_AUDIT") != "1" {
		t.Skip("set AOIKANE_REI_REFERENCE_AUDIT=1 for complete original WAV replay")
	}
	raw, err := os.ReadFile("testdata/rei_wave_original.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		CPU     string
		DLL     string   `json:"dll_sha256"`
		Reports []string `json:"report_sha256"`
		Threads int      `json:"original_threads"`
		AVX512  bool     `json:"avx512_convolutions"`
		Cases   []struct {
			Text, Voice string
			Bytes       int
			SHA256      string
		}
	}
	if err = json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.DLL != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(ref.Reports) != 4 || ref.CPU != "AMD64 Family 25 Model 33 Stepping 0, AuthenticAMD" || ref.Threads != 6 || ref.AVX512 || len(ref.Cases) != 10 {
		t.Fatal("ordinary original WAV profile differs")
	}
	e, err := Open(Options{AssetsDir: "../../assets", OriginalThreads: ref.Threads})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	for i, c := range ref.Cases {
		if (c.Voice != VoiceAkane && c.Voice != VoiceAoi) || c.Bytes < 44 || len(c.SHA256) != 64 {
			t.Fatal("original WAV reference is incomplete")
		}
		t.Run(fmt.Sprintf("%s/case-%d", c.Voice, i), func(t *testing.T) {
			got, err := e.Synthesize(context.Background(), engine.Request{Text: c.Text, Voice: c.Voice})
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != c.Bytes || fmt.Sprintf("%x", sha256.Sum256(got)) != c.SHA256 {
				t.Fatalf("whole WAV differs: bytes=%d SHA256=%x; want %d %s", len(got), sha256.Sum256(got), c.Bytes, c.SHA256)
			}
			t.Logf("whole original WAV identical: %q, %d bytes", c.Text, len(got))
		})
	}
}
