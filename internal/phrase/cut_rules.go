package phrase

import (
	"encoding/json"
	"fmt"
	"io"
)

// CutRules are the loaded BANCHA bancutrcx rules. Groups are alternatives;
// judges are ordered conjunctions. The first matching judge chooses CUT/TIE.
// Strings use UTF-8 rather than the original engine's CP932.
type CutRules struct {
	Version      int                       `json:"version"`
	SourceSHA256 string                    `json:"source_sha256"`
	Groups       map[string][]CutCondition `json:"groups"`
	Judges       []CutJudge                `json:"judges"`
}

type CutCondition struct {
	Kind  string `json:"kind"` // H=POS, K=form, G=type, C=connection, J=group
	ID    uint32 `json:"id,omitempty"`
	Value string `json:"value,omitempty"`
}
type CutJudge struct {
	Cut     bool        `json:"cut"`
	Clauses []CutClause `json:"clauses"`
}
type CutClause struct {
	Group    string `json:"group"`
	Position string `json:"position"` // p=previous, l=current, n=next
	Positive bool   `json:"positive"`
}

func ParseCutRules(reader io.Reader) (*CutRules, error) {
	var r CutRules
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&r); err != nil {
		return nil, err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return nil, fmt.Errorf("trailing cut-rule data")
	}
	if r.Version != 1 || r.SourceSHA256 != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" {
		return nil, fmt.Errorf("unsupported cut-rule asset")
	}
	state := map[string]byte{}
	var visit func(string) error
	visit = func(name string) error {
		group, ok := r.Groups[name]
		if !ok {
			return fmt.Errorf("unknown cut-rule group %q", name)
		}
		if state[name] == 1 {
			return fmt.Errorf("cyclic cut-rule group %q", name)
		}
		if state[name] == 2 {
			return nil
		}
		state[name] = 1
		for _, c := range group {
			switch c.Kind {
			case "H", "K", "G", "C":
			case "J":
				if err := visit(c.Value); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unknown cut-rule condition %q", c.Kind)
			}
		}
		state[name] = 2
		return nil
	}
	for name := range r.Groups {
		if err := visit(name); err != nil {
			return nil, err
		}
	}
	for _, j := range r.Judges {
		for _, c := range j.Clauses {
			if _, ok := r.Groups[c.Group]; !ok {
				return nil, fmt.Errorf("unknown cut-rule group %q", c.Group)
			}
			if c.Position != "p" && c.Position != "l" && c.Position != "n" {
				return nil, fmt.Errorf("invalid cut-rule position %q", c.Position)
			}
		}
	}
	return &r, nil
}

func (r *CutRules) groupMatches(name string, word *Morph) bool {
	if word == nil {
		return false
	}
	for _, c := range r.Groups[name] {
		switch c.Kind {
		case "H":
			if word.POS == c.ID {
				return true
			}
		case "K":
			if word.POS2 == c.ID {
				return true
			}
		case "G":
			if word.POS3 == c.ID {
				return true
			}
		case "J":
			if r.groupMatches(c.Value, word) {
				return true
			}
		case "C":
			// f2a10 compares the complete selected connection string, not its
			// parsed accent-connection rule. An invalid selection yields "".
			connection := ""
			if word.Selected >= 0 && int(word.Selected) < len(word.Cands) {
				connection = word.Cands[word.Selected].Connection
			}
			if connection == c.Value {
				return true
			}
		}
	}
	return false
}

func (r *CutRules) judgeMatches(j CutJudge, prev, current, next *Morph) bool {
	for _, c := range j.Clauses {
		word := current
		if c.Position == "p" {
			word = prev
		} else if c.Position == "n" {
			word = next
		}
		// f1da0 returns AL=0 when all alternatives fail, before consulting
		// +0x29. A matching alternative also fails if +0x29 is zero. Thus a
		// negated clause never matches in this engine, including a missing word.
		if !r.groupMatches(c.Group, word) || !c.Positive {
			return false
		}
	}
	return true
}

func (r *CutRules) shouldCut(prev, current, next *Morph) bool {
	if prev == nil {
		return false
	}
	for _, j := range r.Judges {
		if r.judgeMatches(j, prev, current, next) {
			return j.Cut
		}
	}
	return false
}

// split follows 115110 on [begin,end). S chunks retain the original chunk's
// ordinal, get a serial local to this call, and carry no dependency link.
func (r *CutRules) split(begin, end *Chunk) error {
	if end == nil || end.First == nil {
		return fmt.Errorf("missing cut-range end")
	}
	var serial int32
	for chunk := begin; chunk != nil && chunk != end; chunk = chunk.Next {
		for word := chunk.First; word != nil; word = word.Next {
			if chunk.Next != nil && word == chunk.Next.First {
				break
			}
			var prev *Morph
			if word != chunk.First {
				prev = word.Prev
			}
			next := word.Next
			if next == nil || next.Next == nil || next == end.First {
				next = nil
			}
			if !r.shouldCut(prev, word, next) {
				continue
			}
			if word.Chunk != nil {
				return fmt.Errorf("cut word already starts a chunk")
			}
			inserted := &Chunk{Prev: chunk, Next: chunk.Next, First: word,
				F28: chunk.F28, F2c: serial, F30: 'S', F34: -1}
			serial++
			if inserted.Next != nil {
				inserted.Next.Prev = inserted
			}
			chunk.Next, word.Chunk = inserted, inserted
			chunk = inserted
		}
	}
	return nil
}

// ApplyCut processes each original chunk separately, as 127d70/127ea0 do.
// The input graph must already have its numeral and reading selections.
func (d *Document) ApplyCut(r *CutRules) error {
	if r == nil {
		return nil
	}
	for _, sentence := range d.Sentences {
		end := sentence.Next.Chunk
		for chunk := sentence.Chunk; chunk != end; {
			if chunk == nil {
				return fmt.Errorf("unterminated cut-range")
			}
			next := chunk.Next
			if err := r.split(chunk, next); err != nil {
				return err
			}
			chunk = next
		}
	}
	return nil
}
