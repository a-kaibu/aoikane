package phrase

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/a-kaibu/aoikane/internal/cabocha"
	"github.com/a-kaibu/aoikane/internal/frontend"
	"github.com/a-kaibu/aoikane/internal/numeral"
)

// Frontend carries the reconstructed language assets and builds the word graph
// the native engine holds after its analysis stage.
type Frontend struct {
	Analyzer   *frontend.Analyzer
	Morphology *frontend.Morphology
	Reading    *frontend.ReadingRules
	Codec      *frontend.CP932Codec
	// Bound holds the boundary rules; Predictor is the neural boundary model
	// of the standard voice (nil for voices that do not use one).
	Bound     *BoundRules
	Predictor *BoundModel
	// Numerals reads digit runs and counters (BANCHA numeral pass); nil leaves
	// digits to be read one by one.
	Numerals *numeral.Rules
	// UnknownKana and UnknownAccent read unknown katakana words; without them
	// every unknown word becomes a blank.
	UnknownKana   *frontend.UnknownKanaRules
	UnknownAccent *frontend.UnknownAccentRules
	English       *frontend.EnglishReader
	// Chunker splits the sentence into bunsetsu (CaboCha's CRF chunker); nil
	// keeps the whole sentence as one chunk.
	Chunker *cabocha.CRF
	// Dependency enables SVM links on the chunks before numeral merging.
	// It requires Chunker and the dictionary appropriate to the voice.
	Dependency *cabocha.SVM
	// Cut enables BANCHA sub-chunk splitting after reading selection.
	Cut *CutRules
	// Rendaku voices eligible compound words after BANCHA sub-chunk splitting.
	Rendaku *RendakuRules
	// Kansai selects the dependency-based boundary pass and Kansai accents.
	Kansai *KansaiChunkRules
}

// SplitSentences keeps only the first consecutive sentence-ending mark.
// Intervening horizontal whitespace starts a new run, retaining an empty
// sentence when the next mark arrives. A newline ends unfinished text and
// suppresses immediately following marks.
func SplitSentences(text string) []string {
	var out []string
	var cur []rune
	endedByMark := false
	endedByNewline := false
	flush := func() bool {
		// Leading whitespace is skipped by native sentence extraction, while
		// trailing horizontal whitespace remains an analysed blank word.
		s := strings.TrimLeftFunc(string(cur), unicode.IsSpace)
		cur = cur[:0]
		if strings.TrimSpace(s) == "" {
			return false
		}
		out = append(out, s)
		return true
	}
	for _, r := range text {
		if r == '\n' || r == '\r' {
			emitted := flush()
			if !emitted && len(out) == 0 {
				// Native retains one empty initial sentence for a leading
				// line break, including whitespace-only text before it.
				out = append(out, "")
				emitted = true
			}
			if emitted {
				endedByMark = false
				endedByNewline = true
			}
			continue
		}
		if strings.ContainsRune("。！？!?", r) {
			if endedByMark || endedByNewline {
				continue
			}
			cur = append(cur, r)
			flush()
			endedByMark = true
			continue
		}
		cur = append(cur, r)
		endedByNewline = false
		endedByMark = false
	}
	flush()
	return out
}

// KanaText converts text of several sentences into one formatted reading per
// sentence.
func (f *Frontend) KanaText(text string) ([]string, error) {
	var out []string
	for _, sentence := range SplitSentences(text) {
		if sentence == "" {
			out = append(out, "<S><F>")
			continue
		}
		kana, err := f.Kana(sentence)
		if err != nil {
			return nil, fmt.Errorf("%q: %w", sentence, err)
		}
		out = append(out, kana)
	}
	return out, nil
}

// Kana runs the reconstructed frontend for one sentence. Kansai selects
// dependency-based boundaries and accent combination; otherwise it uses the
// standard boundary predictor. KanaText handles multi-sentence input.
func (f *Frontend) Kana(text string) (string, error) {
	doc, err := f.Graph(text)
	if err != nil {
		return "", err
	}
	if err := doc.attachSingleSentence(); err != nil {
		return "", err
	}
	if err := doc.ApplyCut(f.Cut); err != nil {
		return "", err
	}
	if err := doc.ApplyRendaku(f.Rendaku); err != nil {
		return "", err
	}
	if f.Kansai != nil {
		if err := doc.ApplyKansaiChunkBound(f.Kansai, f.Codec); err != nil {
			return "", err
		}
		doc.ApplyKansaiBoundRules(f.Bound)
	} else if err := doc.ApplyBound(f.Bound); err != nil {
		return "", err
	}
	if f.Predictor != nil {
		if err := f.Predictor.Predict(doc); err != nil {
			return "", err
		}
	}
	doc.Mode, doc.Flag0, doc.Flag21, doc.LimitA, doc.LimitB = 0, true, true, 15, 5
	if f.Kansai != nil {
		doc.Mode = 1
		doc.Flag21 = false
	}
	if err := doc.Build(); err != nil {
		return "", err
	}
	out := ""
	for _, s := range doc.Sentences {
		out += doc.Format(s)
	}
	return out, nil
}

// Graph analyses text and returns the word graph as the engine holds it after
// its BANCHA reading selection.
// Unknown words, which need generated readings, are not supported yet.
func (f *Frontend) Graph(text string) (*Document, error) {
	text = cp932Compatible.Replace(text)
	normalized, err := frontend.NormalizeText(text)
	if err != nil {
		return nil, err
	}
	words, err := f.Analyzer.Analyze(normalized)
	if err != nil {
		return nil, err
	}
	words, generated, err := f.groupWords(words)
	if err != nil {
		return nil, err
	}
	// units are the words after the numeral pass: dictionary words, numeral
	// runs merged into one word and unknown words.
	starts, links, err := f.chunkAnalysisGenerated(words, generated)
	if err != nil {
		return nil, err
	}
	originalStarts := starts
	starts, err = f.prepareNumeralChunks(words, generated, starts)
	if err != nil {
		return nil, err
	}
	var runs []numeralRun
	if f.Numerals != nil {
		runs = f.findNumeralRuns(words, starts)
	}
	var units []unit
	var wordIndices []int
	for i := 0; i < len(words); {
		wordIndices = append(wordIndices, i)
		var run *numeralRun
		for k := range runs {
			if runs[k].start == i {
				run = &runs[k]
			}
		}
		if run != nil {
			surface := run.surface
			units = append(units, unit{numeral: run, surface: surface, chunkStart: starts[run.start],
				lw: frontend.LanguageWord{Surface: surface, Base: surface, POS: 0x1060000}})
			i = run.end
			continue
		}
		w := &words[i]
		if m := generated[i]; m != nil {
			units = append(units, unit{fixed: m, surface: w.Text, chunkStart: starts[i],
				lw: frontend.LanguageWord{Surface: m.Surface, Base: m.Base, POS: m.POS, Form: m.POS2, Type: m.POS3}})
			i++
			continue
		}
		if w.Entry.Unknown {
			u := f.unknownUnit(w.Text)
			u.chunkStart = starts[i]
			units = append(units, u)
			i++
			continue
		}
		lw, err := f.Morphology.LanguageWord(*w)
		if err != nil {
			return nil, err
		}
		units = append(units, unit{word: w, surface: w.Text, lw: lw, chunkStart: starts[i]})
		i++
	}
	links, ordinals, err := contractNumeralChunks(units, wordIndices, len(words), originalStarts, links)
	if err != nil {
		return nil, err
	}
	context := make([]frontend.LanguageWord, len(units))
	surfaces := make([]string, len(units))
	for i, u := range units {
		context[i], surfaces[i] = u.lw, u.surface
		// Dictionary normalization uses U+2212 for minus, while native
		// BANCHA and its reading rules see CP932's U+FF0D representation.
		context[i].Surface = fromCP932Form(context[i].Surface)
		context[i].Base = fromCP932Form(context[i].Base)
	}
	contextText, err := f.Codec.ReadingContextText(text, surfaces)
	if err != nil {
		return nil, err
	}
	var chunks []frontend.ReadingChunk
	var chunkIndices []int
	var chunkWords []frontend.LanguageWord
	if links != nil {
		chunks, chunkIndices, err = readingChunks(units, links, ordinals)
		if err != nil {
			return nil, err
		}
		chunkWords = append(append([]frontend.LanguageWord(nil), context...), frontend.LanguageWord{})
	}
	// BANCHA merges numerals and selects readings chunk by chunk, so the
	// words after the current chunk still hold unmerged numerals: もう before
	// the next chunk's 一本 sees 一 (２モウ).
	chunkEnd := make([]int, len(units))
	for i := len(units) - 1; i >= 0; i-- {
		chunkEnd[i] = i
		if i+1 < len(units) && !units[i+1].chunkStart {
			chunkEnd[i] = chunkEnd[i+1]
		}
	}
	laterNumeral := make([]bool, len(units)+1)
	for i := len(units) - 1; i >= 0; i-- {
		laterNumeral[i] = laterNumeral[i+1] || units[i].numeral != nil
	}
	selectionContext := func(i int) ([]frontend.LanguageWord, string, error) {
		end := chunkEnd[i] + 1
		if !laterNumeral[end] {
			return context, contextText, nil
		}
		ctx := append([]frontend.LanguageWord(nil), context[:end]...)
		surf := append([]string(nil), surfaces[:end]...)
		for _, u := range units[end:] {
			if u.numeral == nil {
				lw := u.lw
				lw.Surface, lw.Base = fromCP932Form(lw.Surface), fromCP932Form(lw.Base)
				ctx, surf = append(ctx, lw), append(surf, u.surface)
				continue
			}
			for k := u.numeral.start; k < u.numeral.end; k++ {
				lw, err := f.Morphology.LanguageWord(words[k])
				if err != nil || words[k].Entry.Unknown {
					lw = frontend.LanguageWord{Surface: words[k].Text, POS: 0xd030000}
				}
				lw.Surface, lw.Base = fromCP932Form(lw.Surface), fromCP932Form(lw.Base)
				ctx, surf = append(ctx, lw), append(surf, words[k].Text)
			}
		}
		ct, err := f.Codec.ReadingContextText(text, surf)
		return ctx, ct, err
	}
	selectReading := func(i int, names []string) (frontend.ReadingSelection, error) {
		var evaluate frontend.ReadingScopeEvaluator
		if chunks != nil {
			evaluate = func(scope frontend.ReadingContext) (bool, error) {
				return f.Reading.EvaluateChunkContext(scope, chunkWords, chunks, chunkIndices[i])
			}
		}
		ctx, ct, err := selectionContext(i)
		if err != nil {
			return frontend.ReadingSelection{}, err
		}
		return f.Reading.SelectReadingWithScopeEvaluator(ctx, i, names, ct, evaluate)
	}
	doc := &Document{chunkLinks: links, chunkOrdinals: ordinals}
	var prev *Morph
	doc.chunkStarts = map[*Morph]bool{}
	var pendingStart bool
	node := func(m *Morph) *Morph {
		if pendingStart {
			doc.chunkStarts[m] = true
		}
		m.F190, m.F194 = 2, 10
		m.Selected = 0
		if prev != nil {
			prev.Next, m.Prev = m, prev
		}
		prev = m
		doc.Morphs = append(doc.Morphs, m)
		return m
	}
	for i, u := range units {
		pendingStart = u.chunkStart
		if u.fixed != nil {
			if len(u.fixed.Cands) > 1 {
				names := make([]string, len(u.fixed.Cands))
				for k, c := range u.fixed.Cands {
					names[k] = c.Reading
				}
				selection, err := selectReading(i, names)
				if err != nil {
					return nil, err
				}
				if selection.Rule >= 0 {
					u.fixed.Cands = u.fixed.Cands[selection.Variant : selection.Variant+1]
				}
			}
			node(u.fixed)
			continue
		}
		if u.numeral != nil {
			m, chain := numeralMorph(u.surface, u.numeral.readings)
			names := make([]string, len(m.Cands))
			for k, c := range m.Cands {
				names[k] = c.Reading
			}
			selection, err := selectReading(i, names)
			if err != nil {
				return nil, fmt.Errorf("numeral %q: %w", u.surface, err)
			}
			if selection.Rule >= 0 {
				m.Cands = m.Cands[selection.Variant : selection.Variant+1]
			}
			node(m)
			if len(chain) > 0 {
				var last *Morph
				for _, c := range chain {
					c.F190, c.F194 = 2, 10
					if last != nil {
						last.Next, c.Prev = c, last
					}
					last = c
				}
				m.ExpFirst, m.ExpLast = chain[0], last
			}
			continue
		}
		w := u.word
		candidates, err := frontend.DictionaryCandidates(w.Entry)
		if err != nil {
			return nil, fmt.Errorf("word %q: %w", w.Text, err)
		}
		names := make([]string, len(candidates))
		for j, c := range candidates {
			names[j] = c.Reading
			if names[j] == "" {
				names[j] = "＊"
			}
		}
		selection, err := selectReading(i, names)
		if err != nil {
			return nil, fmt.Errorf("word %q: %w", w.Text, err)
		}
		m := &Morph{Surface: fromCP932Form(w.Text), Base: fromCP932Form(w.Entry.Surface), POS: context[i].POS, POS2: context[i].Form, POS3: context[i].Type,
			Moras: -1, Up: -1, Down: -1}
		for _, c := range candidates {
			if c.Reading == "" {
				c.Reading = "＊" // the engine labels readingless entries (punctuation) with a placeholder
			}
			m.Cands = append(m.Cands, Candidate{Reading: c.Reading, Pronunciation: c.Pronunciation, Accent: c.Accent, Connection: c.Connection})
		}
		// A matching reading rule makes the BANCHA stage keep only the chosen
		// candidate; otherwise every candidate stays and the first is used.
		if selection.Rule >= 0 {
			m.Cands = m.Cands[selection.Variant : selection.Variant+1]
		} else if _, isSign := f.signOf(w.Text); isSign && len(m.Cands) > 1 && i+1 < len(units) && units[i+1].numeral != nil && w.End == words[units[i+1].numeral.start].Begin {
			sign, _ := f.signOf(w.Text)
			letterBeforeMinus := w.Text == "−" && i > 0 && context[i-1].POS == 0xb040000 && units[i-1].word != nil && units[i-1].word.End == w.Begin
			if i > 0 && units[i-1].numeral != nil || letterBeforeMinus {
				// Between numbers, or directly after an alphabet symbol for
				// minus, native BANCHA keeps the silent first candidate.
				m.Cands = m.Cands[:1]
			} else {
				// a sign in front of a number is read as the sign (マイナス, プラス)
				for _, c := range m.Cands {
					if c.Reading == sign.Read {
						m.Cands = []Candidate{c}
						break
					}
				}
			}
		}
		node(m)
	}
	doc.EOS = node(&Morph{Moras: -1, Up: -1, Down: -1, Selected: 0})
	doc.EOS.Selected = 0
	return doc, nil
}

// attachSingleSentence wraps the morph chain in one sentence with its chunks
// and the end-of-text sentinels. Chunks follow the CRF chunker's boundaries
// when the graph was built with one. Dependency links are retained when
// the frontend was configured with its SVM model.
func (d *Document) attachSingleSentence() error {
	if len(d.Morphs) == 0 || d.EOS == nil {
		return fmt.Errorf("empty document")
	}
	var first, last *Chunk
	count := 0
	for i, m := range d.Morphs {
		if m == d.EOS {
			break
		}
		if i != 0 && !d.chunkStarts[m] {
			continue
		}
		c := &Chunk{First: m, F28: int32(0), F2c: -1, F30: 0x44, F34: -1}
		if d.chunkLinks != nil {
			if count >= len(d.chunkLinks) {
				return fmt.Errorf("CaboCha chunk correspondence changed after word processing")
			}
			c.F34 = d.chunkLinks[count]
		}
		count++
		if d.chunkOrdinals != nil {
			if count > len(d.chunkOrdinals) {
				return fmt.Errorf("chunk ordinal correspondence changed")
			}
			c.F28 = d.chunkOrdinals[count-1]
		} else if last != nil {
			c.F28 = last.F28 + 1
		}
		if last != nil {
			last.Next, c.Prev = c, last
		} else {
			first = c
		}
		m.Chunk = c
		last = c
	}
	if d.chunkLinks != nil && count != len(d.chunkLinks) {
		return fmt.Errorf("CaboCha chunk correspondence changed after word processing")
	}
	endChunk := &Chunk{First: d.EOS, F28: -1, F2c: -1, F34: -1}
	last.Next, endChunk.Prev = endChunk, last
	d.EOS.Chunk = endChunk
	sentinel := &PWord{First: d.EOS, Moras: -1}
	sentence := &Sentence{Chunk: first, PWord: sentinel, Type: 2, End: 10}
	end := &Sentence{Chunk: endChunk, PWord: sentinel, Type: 2, End: 10}
	sentence.Next, end.Prev = end, sentence
	sentinel.Owner = end
	d.Sentences = []*Sentence{sentence}
	d.End = end
	d.PWTail = sentinel
	d.PWCount = 1
	return nil
}

// chunkStarts runs the CRF chunker over the analysed words (as the engine does
// before its BANCHA stage) and reports which words start a chunk.
func (f *Frontend) chunkStarts(words []frontend.Word) (map[int]bool, error) {
	return f.chunkStartsGenerated(words, nil)
}

func (f *Frontend) chunkStartsGenerated(words []frontend.Word, generated map[int]*Morph) (map[int]bool, error) {
	starts, _, err := f.chunkAnalysisGenerated(words, generated)
	return starts, err
}

// unit is a word of the graph under construction.
type unit struct {
	word    *frontend.Word
	numeral *numeralRun
	// fixed is a ready-made word (unknown words).
	fixed   *Morph
	surface string
	lw      frontend.LanguageWord
	// chunkStart is set when the unit starts a bunsetsu chunk.
	chunkStart bool
}

func isKatakana(s string) bool {
	for _, r := range s {
		if !(r >= 'ァ' && r <= 'ヺ' || r == 'ー' || r == '・') {
			return false
		}
	}
	return s != ""
}

// unknownUnit reads an unknown word the way the engine does: katakana keep
// their kana as reading and get the unknown-word accent; anything else (emoji,
// rare kanji) becomes a blank word without reading.
func (f *Frontend) unknownUnit(text string) unit {
	if f.UnknownKana != nil && f.UnknownAccent != nil && isKatakana(text) {
		if r, err := frontend.CompileUnknownKana(text, f.UnknownKana, f.UnknownAccent); err == nil {
			up := 1
			if r.Accent.Up != nil {
				up = *r.Accent.Up
			} else if r.Accent.Down != 1 && r.Accent.Moras != 1 {
				up = 2
			}
			m := &Morph{Surface: text, Base: text, POS: 0x1040000, Moras: -1, Up: -1, Down: -1,
				Cands: []Candidate{{Reading: text, Pronunciation: r.Pronunciation,
					Accent: fmt.Sprintf("%d-%d-%d", up, r.Accent.Down, r.Accent.Moras), Connection: "*"}}}
			return unit{fixed: m, surface: text, lw: frontend.LanguageWord{Surface: text, Base: text, POS: 0x1040000}}
		}
	}
	// The original keeps a single unknown hiragana's surface and empty
	// candidate. a9ea0 later fills its katakana pronunciation with a flat
	// accent; replacing the surface by a blank loses the audible mora.
	if rs := []rune(text); len(rs) == 1 && isHiragana(rs[0]) {
		m := &Morph{Surface: text, POS: 0xd030000, Moras: -1, Up: -1, Down: -1, Cands: []Candidate{{}}}
		return unit{fixed: m, surface: text, lw: frontend.LanguageWord{Surface: text, POS: 0xd030000}}
	}
	m := &Morph{Surface: " ", POS: 0xd030000, Moras: -1, Up: -1, Down: -1, Cands: []Candidate{{}}}
	return unit{fixed: m, surface: " ", lw: frontend.LanguageWord{Surface: " ", POS: 0xd030000}}
}

// cp932Compatible maps characters the engine's code page cannot hold to the
// ones Windows converts them to (minus sign and dashes to full-width forms).
var cp932Compatible = strings.NewReplacer(
	"−", "－", "‐", "－", "‑", "－", "‒", "－", "–", "－",
	"—", "―", " ", " ",
)

// fromCP932Form returns the character the engine's CP932 text holds for the
// dictionary's minus sign (U+2212 reads back as U+FF0D).
func fromCP932Form(s string) string { return strings.ReplaceAll(s, "−", "－") }

// signOf returns the numeral rules' sign entry for a dictionary word.
func (f *Frontend) signOf(text string) (numeral.Word, bool) {
	if f.Numerals == nil {
		return numeral.Word{}, false
	}
	w, ok := f.Numerals.Signs[strings.ReplaceAll(text, "−", "－")]
	return w, ok
}
