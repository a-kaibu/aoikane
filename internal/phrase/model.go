// Package phrase reconstructs the engine's accent-phrase construction: the
// pass of 1aa4f0 that groups bound words into prosodic words, combines their
// accents with the dictionary connection rules and limits their length.
//
// The types mirror the native graph. A document holds a sentence list; every
// sentence has a bunsetsu-chunk list (built by the earlier boundary pass) and a
// prosodic-word list (built here). All lists run through the whole document, so
// the last element of one sentence is delimited by the first element of the
// next, and the final sentence is a sentinel that carries the end-of-text morph.
package phrase

import "fmt"

// Candidate is one dictionary reading with its accent and connection text.
// Strings are UTF-8; the native engine stores CP932.
type Candidate struct {
	Reading, Pronunciation, Accent, Connection string
}

// Origin is an input-derived element (tag, ruby or similar) attached to a
// morph. Only its type is consulted by the phrase pass.
type Origin struct {
	Type int32
}

// Morph is a word of the analysed text (native AIMorph).
type Morph struct {
	Prev, Next *Morph
	// PWord is the prosodic word that starts at this morph (native +0x18).
	PWord *PWord
	// Chunk is the bunsetsu chunk that starts at this morph (native +0x20).
	Chunk *Chunk

	Surface string // +0x78
	Base    string // +0x50

	POS, POS2, POS3 uint32 // +0xb8, +0xc0, +0xc8

	Kind, Kind2 int32  // +0xd0, +0xd4 boundary classification
	Annotation  string // +0xd8 decision label

	// Reading accent cache (+0x100..+0x10c): rise, fall, moras.
	Up, Down, Moras int32

	Cands    []Candidate // +0x110
	Selected int32       // +0x11c

	Origins []Origin // +0x148

	// ExpFirst/ExpLast (+0x28/+0x30) are a detached word chain that replaces the
	// morph's reading when the document is finalised (numerals, symbols).
	ExpFirst, ExpLast *Morph

	// TagPtr/TagLen are the raw tag text pointer and length (+0x160/+0x168).
	TagPtr uint64
	TagLen uint64

	Label170 string // +0x170

	// F190 and F194 are the sentence style fields (+0x190, +0x194).
	F190, F194 int32

	Index int // position in the document chain, set by the loader
}

// Chunk is a bunsetsu chunk produced by the boundary pass (native AIChunk).
type Chunk struct {
	Prev, Next *Chunk
	First      *Morph
	F28, F2c   int32
	F30, F34   int32
}

// PWord is a prosodic word / accent phrase (native AIPWord).
type PWord struct {
	Prev, Next *PWord
	Owner      *Sentence // +0x18: set on the first prosodic word of a sentence
	First      *Morph    // +0x20
	Up, Down   int32     // +0x28, +0x2c
	Moras      int32     // +0x30
	Strength   float32   // +0x34
	Flag38     bool
	Flag39     bool
}

// Sentence is a sentence (native AISent).
type Sentence struct {
	Prev, Next *Sentence
	PWord      *PWord // +0x18 first prosodic word
	Chunk      *Chunk // +0x20 first chunk
	Type       int32  // +0x28
	End        int32  // +0x2c
}

// Document owns the sentences and the prosodic-word list.
type Document struct {
	// Sentences are the real sentences; End is the sentinel that follows them.
	Sentences []*Sentence
	End       *Sentence // +0x220

	PWHead  *PWord // +0x1f8
	PWTail  *PWord // +0x200: sentinel prosodic word
	PWCount int32  // +0x208
	EOS     *Morph // +0x1c0: end-of-text morph

	// Settings of the frontend object.
	Flag0      bool  // +0
	Mode       int32 // +4: language mode (1 for the Kansai dictionary)
	LimitA     int32 // +8
	LimitB     int32 // +0xc
	Flag21     bool  // +0x21
	Morphs     []*Morph
	endOfChain *Morph
	// chunkStarts marks the first word of every bunsetsu chunk (set when the
	// graph is built from text; empty means one chunk).
	chunkStarts map[*Morph]bool
	// chunkOrdinals retains CaboCha ordinals when numeral merging removes chunks.
	chunkOrdinals []int32
	// chunkLinks holds dependencies in those original ordinal coordinates.
	// nil means dependency parsing was not configured.
	chunkLinks []int32
}

func (m *Morph) String() string { return fmt.Sprintf("morph(%q)", m.Surface) }

// silent reports whether the morph carries no reading and no tag text
// (moras < 1 and the tag pointer at +0x160 is empty).
func (m *Morph) silent() bool { return m.Moras < 1 && m.TagPtr == 0 }

// Pronunciation returns the selected candidate's pronunciation.
func (m *Morph) Pronunciation() string {
	if c := m.SelectedCandidate(); c != nil {
		return c.Pronunciation
	}
	return ""
}

// SelectedCandidate returns the selected candidate or nil.
func (m *Morph) SelectedCandidate() *Candidate {
	if m.Selected < 0 || int(m.Selected) >= len(m.Cands) {
		return nil
	}
	return &m.Cands[m.Selected]
}
