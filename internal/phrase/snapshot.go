package phrase

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
)

// Decoder converts native CP932 bytes to UTF-8.
type Decoder interface {
	Decode([]byte) (string, error)
}

// Snapshot is one JSON-lines record written by tools/probe_graph.py.
type Snapshot struct {
	CutBegin  string                 `json:"cut_begin,omitempty"`
	CutEnd    string                 `json:"cut_end,omitempty"`
	Event     string                 `json:"event"`
	Input     string                 `json:"input"`
	Frontend  string                 `json:"frontend"`
	Doc       string                 `json:"doc"`
	Sentences []string               `json:"sentences"`
	End       string                 `json:"end"`
	Nodes     map[string]snapshotRaw `json:"nodes"`
	Text      string                 `json:"text"`
	// Sentence numbers the frontend calls of one captured text (batch captures).
	Sentence int `json:"sentence"`
}

type snapshotRaw struct {
	Kind  string            `json:"kind"`
	Raw   string            `json:"raw"`
	Str   map[string]string `json:"str"`
	Cands []struct {
		S []*string `json:"s"`
	} `json:"cands"`
}

// ReadSnapshots reads all records of a probe file.
func ReadSnapshots(path string) ([]Snapshot, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := bufio.NewReaderSize(f, 1<<20)
	var out []Snapshot
	for {
		line, err := r.ReadBytes('\n')
		if len(strings.TrimSpace(string(line))) > 0 {
			var s Snapshot
			if jerr := json.Unmarshal(line, &s); jerr != nil {
				return nil, jerr
			}
			out = append(out, s)
		}
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
	}
}

// FormatText decodes the CP932 text of a "format" record.
func (s Snapshot) FormatText(dec Decoder) (string, error) {
	b, err := hex.DecodeString(s.Text)
	if err != nil {
		return "", err
	}
	return dec.Decode(b)
}

type loader struct {
	s      Snapshot
	dec    Decoder
	raw    map[uint64][]byte
	morphs map[uint64]*Morph
	chunks map[uint64]*Chunk
	pwords map[uint64]*PWord
	sents  map[uint64]*Sentence
}

func u64(b []byte, o int) uint64 { return binary.LittleEndian.Uint64(b[o:]) }
func i32(b []byte, o int) int32  { return int32(binary.LittleEndian.Uint32(b[o:])) }
func u32(b []byte, o int) uint32 { return binary.LittleEndian.Uint32(b[o:]) }

func parseAddr(s string) (uint64, error) {
	var v uint64
	_, err := fmt.Sscanf(s, "0x%x", &v)
	return v, err
}

// Load converts a graph snapshot into a Document.
func (s Snapshot) Load(dec Decoder) (*Document, error) {
	l := &loader{s: s, dec: dec, raw: map[uint64][]byte{},
		morphs: map[uint64]*Morph{}, chunks: map[uint64]*Chunk{}, pwords: map[uint64]*PWord{}, sents: map[uint64]*Sentence{}}
	kinds := map[uint64]string{}
	for k, n := range s.Nodes {
		a, err := parseAddr(k)
		if err != nil {
			return nil, err
		}
		kinds[a] = n.Kind
		if n.Raw != "" {
			b, err := hex.DecodeString(n.Raw)
			if err != nil {
				return nil, err
			}
			l.raw[a] = b
		}
	}
	// allocate nodes first so that links can be resolved
	for a, kind := range kinds {
		switch kind {
		case "morph":
			l.morphs[a] = &Morph{}
		case "chunk":
			l.chunks[a] = &Chunk{}
		case "word":
			l.pwords[a] = &PWord{}
		case "sent":
			l.sents[a] = &Sentence{}
		}
	}
	m := func(a uint64) *Morph { return l.morphs[a] }
	for a, mo := range l.morphs {
		b := l.raw[a]
		if len(b) < 0x1c0 {
			return nil, fmt.Errorf("morph %#x truncated", a)
		}
		node := s.Nodes[fmt.Sprintf("0x%x", a)]
		mo.Prev, mo.Next = m(u64(b, 8)), m(u64(b, 0x10))
		mo.PWord, mo.Chunk = l.pwords[u64(b, 0x18)], l.chunks[u64(b, 0x20)]
		mo.ExpFirst, mo.ExpLast = m(u64(b, 0x28)), m(u64(b, 0x30))
		for _, off := range []int{8, 0x10} {
			if p := u64(b, off); p != 0 && m(p) == nil {
				return nil, fmt.Errorf("morph %#x links to unknown node %#x", a, p)
			}
		}
		mo.POS, mo.POS2, mo.POS3 = u32(b, 0xb8), u32(b, 0xc0), u32(b, 0xc8)
		mo.Kind, mo.Kind2 = i32(b, 0xd0), i32(b, 0xd4)
		mo.Up, mo.Down, mo.Moras = i32(b, 0x100), i32(b, 0x104), i32(b, 0x108)
		mo.Selected = i32(b, 0x11c)
		if n := u64(b, 0x150) - u64(b, 0x148); n != 0 {
			return nil, fmt.Errorf("morph %#x has input-origin elements, which are not captured", a)
		}
		mo.TagPtr, mo.TagLen = u64(b, 0x160), u64(b, 0x168)
		mo.F190, mo.F194 = i32(b, 0x190), i32(b, 0x194)
		var err error
		if mo.Base, err = l.str(node.Str["80"]); err != nil {
			return nil, err
		}
		if mo.Surface, err = l.str(node.Str["120"]); err != nil {
			return nil, err
		}
		if mo.Annotation, err = l.str(node.Str["216"]); err != nil {
			return nil, err
		}
		if mo.Label170, err = l.str(node.Str["368"]); err != nil {
			return nil, err
		}
		for _, c := range node.Cands {
			var cand Candidate
			dst := []*string{&cand.Reading, &cand.Pronunciation, &cand.Accent, &cand.Connection}
			for i, d := range dst {
				if i < len(c.S) && c.S[i] != nil {
					if *d, err = l.str(*c.S[i]); err != nil {
						return nil, err
					}
				}
			}
			mo.Cands = append(mo.Cands, cand)
		}
	}
	for a := range l.chunks {
		if len(l.raw[a]) < 0x40 {
			return nil, fmt.Errorf("chunk %#x was not captured", a)
		}
	}
	for a := range l.pwords {
		if len(l.raw[a]) < 0x40 {
			return nil, fmt.Errorf("prosodic word %#x was not captured", a)
		}
	}
	for a := range l.sents {
		if len(l.raw[a]) < 0x40 {
			return nil, fmt.Errorf("sentence %#x was not captured", a)
		}
	}
	for a, c := range l.chunks {
		b := l.raw[a]
		c.Prev, c.Next, c.First = l.chunks[u64(b, 8)], l.chunks[u64(b, 0x10)], m(u64(b, 0x20))
		c.F28, c.F2c, c.F30, c.F34 = i32(b, 0x28), i32(b, 0x2c), i32(b, 0x30), i32(b, 0x34)
	}
	for a, p := range l.pwords {
		b := l.raw[a]
		p.Prev, p.Next = l.pwords[u64(b, 8)], l.pwords[u64(b, 0x10)]
		p.Owner, p.First = l.sents[u64(b, 0x18)], m(u64(b, 0x20))
		p.Up, p.Down, p.Moras = i32(b, 0x28), i32(b, 0x2c), i32(b, 0x30)
		p.Strength = math.Float32frombits(u32(b, 0x34))
		p.Flag38, p.Flag39 = b[0x38] != 0, b[0x39] != 0
	}
	for a, st := range l.sents {
		b := l.raw[a]
		st.Prev, st.Next = l.sents[u64(b, 8)], l.sents[u64(b, 0x10)]
		st.PWord, st.Chunk = l.pwords[u64(b, 0x18)], l.chunks[u64(b, 0x20)]
		st.Type, st.End = i32(b, 0x28), i32(b, 0x2c)
	}
	doc := &Document{}
	fr, err := hex.DecodeString(s.Frontend)
	if err != nil || len(fr) < 0x30 {
		return nil, fmt.Errorf("invalid frontend record")
	}
	doc.Flag0, doc.Mode, doc.LimitA, doc.LimitB, doc.Flag21 = fr[0] != 0, i32(fr, 4), i32(fr, 8), i32(fr, 0xc), fr[0x21] != 0
	d, err := hex.DecodeString(s.Doc)
	if err != nil || len(d) < 0x230 {
		return nil, fmt.Errorf("invalid document record")
	}
	doc.EOS = m(u64(d, 0x1c0))
	doc.PWHead, doc.PWTail, doc.PWCount = l.pwords[u64(d, 0x1f8)], l.pwords[u64(d, 0x200)], i32(d, 0x208)
	doc.End = l.sents[u64(d, 0x220)]
	for _, a := range s.Sentences {
		addr, err := parseAddr(a)
		if err != nil {
			return nil, err
		}
		doc.Sentences = append(doc.Sentences, l.sents[addr])
	}
	// The main chain is the one that ends at the end-of-text morph; detached
	// expansion chains are numbered after it.
	var main *Morph
	for _, mo := range l.morphs {
		if mo.Prev == nil {
			for x := mo; x != nil; x = x.Next {
				if x == doc.EOS {
					main = mo
				}
			}
		}
	}
	if main == nil {
		return nil, fmt.Errorf("no morph chain reaches the end-of-text morph")
	}
	for mo := main; mo != nil; mo = mo.Next {
		mo.Index = len(doc.Morphs)
		doc.Morphs = append(doc.Morphs, mo)
	}
	return doc, nil
}

func (l *loader) str(h string) (string, error) {
	if h == "" {
		return "", nil
	}
	b, err := hex.DecodeString(h)
	if err != nil {
		return "", err
	}
	return l.dec.Decode(b)
}
