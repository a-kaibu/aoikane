package phrase

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// BoundRules are the boundary rules of the BOUND pass (boundlrcx): ordered
// groups of patterns, each naming a boundary kind. A pattern matches a morph
// when one of its current-word conditions matches at the morph, one previous-
// word condition matches before it and one next-word condition after it.
type BoundRules struct {
	conds  map[int]*boundCond
	Groups []BoundGroup
}

// BoundGroup is one "Boundary:" block.
type BoundGroup struct {
	Name     string
	Kind     int32
	Patterns []BoundPattern
}

// BoundPattern is one "Pattern:" entry. Condition IDs reference the table.
type BoundPattern struct {
	Label               string
	Current, Prev, Next []int
}

type boundCond struct {
	op       string // "leaf", "and", "or", "seq", "true"
	negate   bool
	field    string
	operator string // EQ or IN
	strs     []string
	ints     []uint32
	children []int
}

// boundKindIDs maps boundary names to the engine's kind identifiers (table at
// 180391310).
var boundKindIDs = map[string]int32{
	"係り受け": 1, "節境界": 2, "非修飾境界２": 3, "非修飾境界３": 4, "複合語内": 5, "文境界": 6,
	"息継ぎ境界２": 7, "息継ぎ境界３": 8, "並列読点境界３": 9, "感動節": 10, "動詞基本形息継ぎ": 11,
	"非境界": 12, "長休止記号境界": 13, "短休止記号境界": 14, "強調境界": 15,
	"副次アクセント句境界候補": 16, "アクセント句境界": 17, "上書きアクセント句境界": 18,
	"PauseCandidate": 19, "文末休止": 20, "複アクセント分割": 21, "読点境界": 22, "ポーズ削除": 23,
	"AIカナ": 24, "AIカナ(境界記号のみ)": 25, "NoPauseBoundary": 26, "NoPauseAccentBoundary": 27,
	"Predict1": 28, "Predict2": 29, "境界タグ": 30,
	"文末タグＦ": 31, "文末タグＲ": 32, "文末タグＷ": 33, "文末タグＨ": 34, "文末タグＡ": 35,
	"文末タグＪ": 36, "文末タグＭ": 37, "文末タグＮ": 38, "文末タグＣ": 39, "文末タグＴ": 40,
	"スタイルタグ感動": 41,
}

// ParseBoundRules reads the UTF-8 form of a boundlrcx table.
func ParseBoundRules(r io.Reader) (*BoundRules, error) {
	ss := map[int]string{}
	ms := map[int][]int{}
	mi := map[int][]uint32{}
	rules := &BoundRules{conds: map[int]*boundCond{}}
	var group *BoundGroup
	var pattern *BoundPattern
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1<<16), 1<<22)
	line := 0
	fail := func(format string, args ...any) (*BoundRules, error) {
		return nil, fmt.Errorf("boundary rules line %d: %s", line, fmt.Sprintf(format, args...))
	}
	ref := func(token, prefix string) (int, error) {
		if !strings.HasPrefix(token, prefix+":") {
			return 0, fmt.Errorf("expected %s reference, got %q", prefix, token)
		}
		return strconv.Atoi(token[len(prefix)+1:])
	}
	for scanner.Scan() {
		line++
		raw := strings.TrimRight(scanner.Text(), "\r")
		text := strings.TrimSpace(raw)
		if text == "" || strings.HasPrefix(text, ";") {
			continue
		}
		parts := strings.Fields(text)
		switch {
		case strings.HasPrefix(raw, "SS:"):
			id, err := ref(parts[0], "SS")
			if err != nil {
				return fail("%v", err)
			}
			sep := strings.IndexAny(raw, " \t")
			if sep < 0 {
				return fail("invalid string table entry")
			}
			ss[id] = strings.Trim(raw[sep+1:], " \t")
		case strings.HasPrefix(raw, "MS:"):
			id, err := ref(parts[0], "MS")
			if err != nil {
				return fail("%v", err)
			}
			for _, tok := range parts[1:] {
				v, err := ref(tok, "SS")
				if err != nil {
					return fail("%v", err)
				}
				ms[id] = append(ms[id], v)
			}
		case strings.HasPrefix(raw, "MI:"):
			id, err := ref(parts[0], "MI")
			if err != nil {
				return fail("%v", err)
			}
			for _, tok := range parts[1:] {
				v, err := strconv.ParseUint(tok, 0, 32)
				if err != nil {
					return fail("%v", err)
				}
				mi[id] = append(mi[id], uint32(v))
			}
		case strings.HasPrefix(raw, "CD:"):
			id, err := ref(parts[0], "CD")
			if err != nil {
				return fail("%v", err)
			}
			c, err := parseBoundCond(parts[1:], ss, ms, mi)
			if err != nil {
				return fail("%v", err)
			}
			rules.conds[id] = c
		case strings.HasPrefix(text, "{ Boundary:"):
			name := strings.TrimPrefix(text, "{ Boundary:")
			rules.Groups = append(rules.Groups, BoundGroup{Name: name, Kind: boundKindIDs[name]})
			group = &rules.Groups[len(rules.Groups)-1]
		case strings.HasPrefix(text, "{ Pattern:"):
			if group == nil {
				return fail("pattern outside a boundary block")
			}
			group.Patterns = append(group.Patterns, BoundPattern{Label: strings.TrimPrefix(text, "{ Pattern:")})
			pattern = &group.Patterns[len(group.Patterns)-1]
		case text == "}":
			if pattern != nil {
				pattern = nil
			} else {
				group = nil
			}
		case len(parts) == 2 && (parts[0] == "C" || parts[0] == "P" || parts[0] == "N"):
			if pattern == nil {
				return fail("condition outside a pattern")
			}
			id, err := ref(parts[1], "CD")
			if err != nil {
				return fail("%v", err)
			}
			switch parts[0] {
			case "C":
				pattern.Current = append(pattern.Current, id)
			case "P":
				pattern.Prev = append(pattern.Prev, id)
			default:
				pattern.Next = append(pattern.Next, id)
			}
		default:
			return fail("unsupported line %q", text)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	for _, g := range rules.Groups {
		if g.Kind == 0 {
			return nil, fmt.Errorf("unknown boundary kind %q", g.Name)
		}
		for _, p := range g.Patterns {
			for _, list := range [][]int{p.Current, p.Prev, p.Next} {
				for _, id := range list {
					if _, ok := rules.conds[id]; !ok {
						return nil, fmt.Errorf("pattern %q references missing CD:%d", p.Label, id)
					}
				}
			}
		}
	}
	return rules, nil
}

func parseBoundCond(args []string, ss map[int]string, ms map[int][]int, mi map[int][]uint32) (*boundCond, error) {
	c := &boundCond{}
	if len(args) > 0 && args[0] == "!" {
		c.negate = true
		args = args[1:]
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("empty condition")
	}
	switch args[0] {
	case "TRUE":
		c.op = "true"
		return c, nil
	case "ACCENT":
		// ACCENT [0|1] compares the word's accentedness with the argument
		// (default 0, i.e. "unaccented").
		c.op = "accent"
		if len(args) > 1 {
			v, err := strconv.Atoi(args[1])
			if err != nil || v < 0 || v > 1 {
				return nil, fmt.Errorf("invalid ACCENT argument %q", args[1])
			}
			c.ints = []uint32{uint32(v)}
		}
		return c, nil
	case "&", "|", "+":
		c.op = map[string]string{"&": "and", "|": "or", "+": "seq"}[args[0]]
		for _, tok := range args[1:] {
			if !strings.HasPrefix(tok, "CD:") {
				return nil, fmt.Errorf("expected CD reference, got %q", tok)
			}
			id, err := strconv.Atoi(tok[3:])
			if err != nil {
				return nil, err
			}
			c.children = append(c.children, id)
		}
		return c, nil
	}
	if len(args) != 3 {
		return nil, fmt.Errorf("invalid condition %v", args)
	}
	op := args[0]
	if strings.HasPrefix(op, "!") {
		c.negate = !c.negate
		op = op[1:]
	}
	integer := strings.HasSuffix(op, "_I")
	op = strings.TrimSuffix(op, "_I")
	if op != "EQ" && op != "IN" {
		return nil, fmt.Errorf("unsupported operator %q", args[0])
	}
	c.op, c.field, c.operator = "leaf", args[1], op
	switch c.field {
	case "POS", "POS_L1", "POS_L2", "POS_L3", "CTYPE", "CFORM", "KANJI", "BASE", "PRON", "READ":
	default:
		return nil, fmt.Errorf("unsupported field %q", c.field)
	}
	if integer {
		if op == "EQ" {
			v, err := strconv.ParseUint(args[2], 0, 32)
			if err != nil {
				return nil, err
			}
			c.ints = []uint32{uint32(v)}
		} else {
			if !strings.HasPrefix(args[2], "MI:") {
				return nil, fmt.Errorf("expected MI reference")
			}
			id, _ := strconv.Atoi(args[2][3:])
			set, ok := mi[id]
			if !ok {
				return nil, fmt.Errorf("missing MI:%d", id)
			}
			c.ints = set
		}
		return c, nil
	}
	var ids []int
	if op == "EQ" {
		id, err := strconv.Atoi(strings.TrimPrefix(args[2], "SS:"))
		if err != nil {
			return nil, err
		}
		ids = []int{id}
	} else {
		id, err := strconv.Atoi(strings.TrimPrefix(args[2], "MS:"))
		if err != nil {
			return nil, err
		}
		set, ok := ms[id]
		if !ok {
			return nil, fmt.Errorf("missing MS:%d", id)
		}
		ids = set
	}
	for _, id := range ids {
		s, ok := ss[id]
		if !ok {
			return nil, fmt.Errorf("missing SS:%d", id)
		}
		c.strs = append(c.strs, s)
	}
	return c, nil
}

// width is the number of morphs a condition consumes.
func (r *BoundRules) width(c *boundCond) int {
	if c.op == "seq" {
		return len(c.children)
	}
	return 1
}

// matchLeaf evaluates a condition on a single morph.
func (r *BoundRules) matchOne(c *boundCond, m *Morph) bool {
	var matched bool
	switch c.op {
	case "true":
		matched = true
	case "accent":
		want := len(c.ints) != 0 && c.ints[0] == 1
		if m.Moras < m.Down || m.Down == 0 {
			want = !want
		}
		matched = want
	case "leaf":
		matched = c.leaf(m)
		return matched != c.negate
	case "and", "or":
		matched = true // an empty list yields true in the native evaluator
		for _, id := range c.children {
			matched = r.matchOne(r.conds[id], m)
			if c.op == "or" == matched {
				break
			}
		}
	}
	return matched != c.negate
}

func (c *boundCond) leaf(m *Morph) bool {
	switch c.field {
	case "POS", "POS_L1", "POS_L2", "POS_L3", "CTYPE", "CFORM":
		value, mask := m.POS, uint32(0xffffffff)
		switch c.field {
		case "POS_L1":
			mask = 0xff000000
		case "POS_L2":
			mask = 0xffff0000
		case "POS_L3":
			mask = 0xffffff00
		case "CTYPE":
			value = m.POS3
		case "CFORM":
			value = m.POS2
		}
		value &= mask
		for _, v := range c.ints {
			if v == value {
				return true
			}
		}
		return false
	}
	var value string
	switch c.field {
	case "KANJI":
		value = m.Surface
	case "BASE":
		value = m.Base
	case "READ":
		if cand := m.SelectedCandidate(); cand != nil {
			value = cand.Reading
		}
	case "PRON":
		value = m.Pronunciation()
	}
	for _, s := range c.strs {
		if s == value {
			return true
		}
	}
	return false
}

// matchSeq evaluates a sequence condition over consecutive morphs from start.
func (r *BoundRules) matchSeq(c *boundCond, words []*Morph, start int) bool {
	matched := false
	idx := start
	if idx >= 0 {
		for _, id := range c.children {
			if idx >= len(words) {
				matched = false
				break
			}
			matched = r.matchOne(r.conds[id], words[idx])
			if !matched {
				break
			}
			idx++
		}
	}
	return matched != c.negate
}

// matchPattern is 1365d0.
func (r *BoundRules) matchPattern(p *BoundPattern, words []*Morph, idx int) bool {
	consumed := 0
	if len(p.Current) != 0 {
		ok := false
		for _, id := range p.Current {
			c := r.conds[id]
			w := r.width(c)
			var v bool
			if w == 1 {
				v = r.matchOne(c, words[idx])
			} else {
				v = r.matchSeq(c, words, idx)
			}
			if v {
				consumed, ok = w-1, true
				break
			}
		}
		if !ok {
			return false
		}
	}
	if len(p.Prev) != 0 {
		if idx == 0 {
			return false
		}
		ok := false
		for _, id := range p.Prev {
			c := r.conds[id]
			w := r.width(c)
			var v bool
			if w == 1 {
				v = r.matchOne(c, words[idx-1])
			} else {
				v = r.matchSeq(c, words, idx-w)
			}
			if v {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	if len(p.Next) != 0 {
		at := idx + consumed
		if at+1 >= len(words) {
			return false
		}
		for _, id := range p.Next {
			c := r.conds[id]
			var v bool
			if r.width(c) == 1 {
				v = r.matchOne(c, words[at+1])
			} else {
				v = r.matchSeq(c, words, at+1)
			}
			if v {
				return true
			}
		}
		return false
	}
	return true
}

// Match returns the boundary kind and pattern label that apply to the morph at
// idx of the sentence's morph list (kind 0 when no pattern matches).
func (r *BoundRules) Match(words []*Morph, idx int) (int32, string) {
	for gi := range r.Groups {
		g := &r.Groups[gi]
		for pi := range g.Patterns {
			if r.matchPattern(&g.Patterns[pi], words, idx) {
				return g.Kind, g.Patterns[pi].Label
			}
		}
	}
	return 0, ""
}

// ApplyBound is 135c60: it refreshes the reading cache of every morph and
// assigns the boundary kind (kinds below 0x1f) or sentence-end style (kind2)
// of the first matching pattern.
func (d *Document) ApplyBound(rules *BoundRules) error {
	var words []*Morph
	for _, m := range d.Morphs {
		if m == d.EOS {
			break
		}
		words = append(words, m)
	}
	for i, m := range words {
		if err := updateReading(m); err != nil {
			return err
		}
		kind, label := rules.Match(words, i)
		switch {
		case kind == 0:
		case kind < 0x1f:
			if m.Kind == 0 {
				m.Kind = kind
				m.Annotation = label
			}
		default:
			m.Kind2 = kind
			m.Annotation = label
		}
	}
	return nil
}
