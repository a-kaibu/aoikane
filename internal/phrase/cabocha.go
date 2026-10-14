package phrase

import (
	"fmt"
	"strings"

	"github.com/a-kaibu/aoikane/internal/cabocha"
	"github.com/a-kaibu/aoikane/internal/frontend"
)

// cabochaInput retains the correspondence with words before numeral merging.
// The Selector consumes the four POS columns and the type/form columns only;
// readings and accents continue through the original word graph path.
type cabochaInput struct {
	Tokens      []cabocha.Token
	Rows        [][]string
	WordIndices []int
}

func morphologyIDName(table []frontend.MorphologyValue, id uint32) string {
	for _, value := range table {
		if value.ID == id {
			return value.Name
		}
	}
	if len(table) > 0 {
		return table[0].Name
	}
	return ""
}

func (f *Frontend) cabochaInputGenerated(words []frontend.Word, generated map[int]*Morph) (*cabochaInput, error) {
	input := new(cabochaInput)
	for i, w := range words {
		lw := frontend.LanguageWord{Surface: w.Text, Base: w.Text, POS: 0xd030000}
		if m := generated[i]; m != nil {
			lw = frontend.LanguageWord{Surface: w.Text, Base: m.Base, POS: m.POS, Type: m.POS3, Form: m.POS2}
		} else if !w.Entry.Unknown {
			var err error
			lw, err = f.Morphology.LanguageWord(w)
			if err != nil {
				return nil, err
			}
		} else if isKatakana(w.Text) {
			lw.POS = 0x1040000
		}
		if lw.POS&0xffff0000 == 0xb080000 {
			continue
		}
		surface := w.Text
		if w.Entry.Unknown && generated[i] == nil && !isKatakana(w.Text) {
			surface = " "
		}
		encoded, err := f.Codec.Encode(surface)
		if err != nil {
			encoded = []byte(surface)
		}
		name, ok := cabocha.POSNames[lw.POS]
		if !ok {
			return nil, fmt.Errorf("unknown CaboCha POS %08x", lw.POS)
		}
		encodedPOS, err := f.Codec.Encode(name)
		if err != nil {
			return nil, err
		}
		// Match the CP932 representation used by the native CSV serializer.
		surface, err = f.Codec.Decode(encoded)
		if err != nil {
			return nil, err
		}
		fields := strings.Split(name, "-")
		for len(fields) < 4 {
			fields = append(fields, "*")
		}
		fields = append(fields, morphologyIDName(f.Morphology.Types, lw.Type), morphologyIDName(f.Morphology.Forms, lw.Form))
		input.Tokens = append(input.Tokens, cabocha.Token{Surface: surface, Normalized: surface, Features: fields})
		input.Rows = append(input.Rows, []string{string(encoded), string(encodedPOS)})
		input.WordIndices = append(input.WordIndices, i)
	}
	return input, nil
}

func (input *cabochaInput) spans(model *cabocha.CRF) []cabocha.ChunkSpan {
	if len(input.Tokens) == 0 {
		return nil
	}
	starts := []int{0}
	for i, tag := range model.Tag(input.Rows) {
		if i > 0 && tag == "B" {
			starts = append(starts, i)
		}
	}
	result := make([]cabocha.ChunkSpan, len(starts))
	for i, start := range starts {
		end := len(input.Tokens)
		if i+1 < len(starts) {
			end = starts[i+1]
		}
		result[i] = cabocha.ChunkSpan{Start: start, Size: end - start}
	}
	return result
}

func (f *Frontend) chunkAnalysisGenerated(words []frontend.Word, generated map[int]*Morph) (map[int]bool, []int32, error) {
	starts := map[int]bool{}
	if f.Chunker == nil {
		if f.Dependency != nil {
			return nil, nil, fmt.Errorf("CaboCha dependency model requires a chunker")
		}
		return starts, nil, nil
	}
	input, err := f.cabochaInputGenerated(words, generated)
	if err != nil {
		return nil, nil, err
	}
	spans := input.spans(f.Chunker)
	for _, span := range spans {
		starts[input.WordIndices[span.Start]] = true
	}
	if f.Dependency == nil {
		return starts, nil, nil
	}
	selection, err := cabocha.SelectIPA(input.Tokens, spans)
	if err != nil {
		return nil, nil, err
	}
	deps, err := f.Dependency.Dependencies(selection, f.Codec)
	if err != nil {
		return nil, nil, err
	}
	links := make([]int32, len(deps))
	for i, dep := range deps {
		links[i] = int32(dep.Link)
	}
	return starts, links, nil
}
