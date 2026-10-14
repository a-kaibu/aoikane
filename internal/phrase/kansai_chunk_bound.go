package phrase

import (
	"github.com/a-kaibu/aoikane/internal/frontend"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type KansaiChunkRule struct {
	Kind      int32   `json:"kind"`
	POS       []int32 `json:"pos"`
	Distance  []int32 `json:"distance"`
	Between   []int32 `json:"between"`
	Inclusive []int32 `json:"inclusive"`
}
type KansaiChunkRules struct {
	Version      int               `json:"version"`
	SourceSHA256 string            `json:"source_sha256"`
	Rules        []KansaiChunkRule `json:"rules"`
}

func ParseKansaiChunkRules(r io.Reader) (*KansaiChunkRules, error) {
	var rules KansaiChunkRules
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&rules); err != nil {
		return nil, err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("trailing Kansai chunk rules")
	}
	if rules.Version != 1 || rules.SourceSHA256 != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(rules.Rules) == 0 {
		return nil, fmt.Errorf("invalid Kansai chunk rules")
	}
	for _, r := range rules.Rules {
		if r.Kind < 1 || r.Kind > 30 {
			return nil, fmt.Errorf("invalid Kansai boundary kind")
		}
		for _, v := range [][]int32{r.Distance, r.Between, r.Inclusive} {
			if len(v) > 2 || len(v) == 2 && v[0] > v[1] {
				return nil, fmt.Errorf("invalid Kansai boundary range")
			}
		}
	}
	return &rules, nil
}
func kansaiRange(values []int32, value int32) bool {
	return len(values) == 0 || len(values) == 1 && values[0] == value || len(values) == 2 && values[0] <= value && value <= values[1]
}

// kansaiMoras follows 1962d0 and its distinct ten-entry small-kana table.
// The original scans CP932 two-byte units, including a leading small kana.
func kansaiMoras(text string, codec *frontend.CP932Codec) (int32, error) {
	if codec == nil {
		return 0, fmt.Errorf("missing Kansai CP932 codec")
	}
	if strings.ContainsRune(text, 0) {
		return 0, fmt.Errorf("NUL in Kansai pronunciation")
	}
	data, err := codec.Encode(text)
	if err != nil {
		return 0, err
	}
	small := func(i int) bool {
		if i+1 >= len(data) {
			return false
		}
		switch uint16(data[i])<<8 | uint16(data[i+1]) {
		case 0x8340, 0x8342, 0x8344, 0x8346, 0x8348, 0x8383, 0x8385, 0x8387, 0x838e, 0x814b:
			return true
		}
		return false
	}
	var n int32
	for i := 0; i+1 < len(data); {
		if data[i] == 0x81 && (data[i+1] == 0x41 || data[i+1] == 0x42) {
			i += 2
			continue
		}
		if !small(i) && small(i+2) {
			i += 4
			if small(i) {
				i += 2
			}
		} else {
			i += 2
		}
		n++
	}
	return n, nil
}
func kansaiPronunciation(m *Morph) string {
	var b strings.Builder
	b.WriteString(m.Pronunciation())
	for x := m.ExpFirst; x != nil; x = x.Next {
		b.WriteString(x.Pronunciation())
		if x == m.ExpLast {
			break
		}
	}
	return b.String()
}

// ApplyKansaiChunkBound reconstructs 137520. Split S chunks are accumulated
// into their preceding dependency chunk before distance/mora rules are applied.
func (d *Document) ApplyKansaiChunkBound(rules *KansaiChunkRules, codec *frontend.CP932Codec) error {
	if rules == nil {
		return fmt.Errorf("missing Kansai chunk-bound rules")
	}
	if len(d.Sentences) == 0 {
		return fmt.Errorf("missing Kansai sentence chunks")
	}
	type group struct {
		chunk *Chunk
		moras int32
		pos   uint32
	}
	var groups []group
	for c := d.Sentences[0].Chunk; c != nil && c.Next != nil && c.Next.First != nil; c = c.Next {
		if c.First == nil {
			return fmt.Errorf("missing Kansai chunk word")
		}
		var moras int32
		for m := c.First; m != nil && m != c.Next.First; m = m.Next {
			n, err := kansaiMoras(kansaiPronunciation(m), codec)
			if err != nil {
				return err
			}
			moras += n
		}
		tail := c.Next.First.Prev
		if tail == nil {
			return fmt.Errorf("missing Kansai chunk tail")
		}
		if c.F30 == 'S' && len(groups) > 0 {
			g := &groups[len(groups)-1]
			g.moras += moras
			g.pos = tail.POS
		} else {
			groups = append(groups, group{c, moras, tail.POS})
		}
	}
	for i := 0; i+1 < len(groups); i++ {
		previous, next := groups[i], groups[i+1]
		link := previous.chunk.F34
		distance := int32(-1)
		for c := previous.chunk; c != nil && c.Next != nil; c = c.Next {
			if c.F30 != 'S' {
				distance++
			}
			if c.F28 == link {
				break
			}
			if c.Next.Next == nil {
				distance = -1
			}
		}
		between := int32(0)
		j := i + 1
		for ; j < len(groups) && groups[j].chunk.F28 < link; j++ {
			between += groups[j].moras
		}
		inclusive := between
		if j < len(groups) && groups[j].chunk.F28 == link {
			inclusive += groups[j].moras
		}
		target := next.chunk.First.Prev
		if target == nil {
			return fmt.Errorf("missing Kansai boundary target")
		}
		for _, rule := range rules.Rules {
			posOK := len(rule.POS) == 0
			for _, p := range rule.POS {
				if uint32(p) == previous.pos {
					posOK = true
				}
			}
			if posOK && kansaiRange(rule.Distance, distance) && kansaiRange(rule.Between, between) && kansaiRange(rule.Inclusive, inclusive) {
				target.Kind = rule.Kind
				break
			}
			if distance > 1 {
				target.Kind = 19
			}
		}
		if (previous.chunk.F30 == 'P' || previous.chunk.F30 == 'A') && target.Kind != 3 && target.Kind != 4 {
			target.Kind = 19
			if previous.pos == 0xb020000 {
				target.Kind = 9
			}
		}
	}
	return nil
}
