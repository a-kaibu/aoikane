package frontend

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ReadingRules retains table identifiers and file order. Rule indices are the
// zero-based indices returned by the inspected native reading selector.
type ReadingRules struct {
	Conditions map[int]ReadingPredicate
	Rules      []ReadingRule
	// SkipPhraseScopes makes rules that need the bunsetsu/dependency graph
	// (P[0:0], N[0:0], N[0:-1]) not match instead of failing when the caller
	// supplies neither a graph evaluator nor whole-context text. Production
	// synthesis supplies both and does not enable this legacy fallback.
	SkipPhraseScopes bool
}

type ReadingPredicate struct {
	Kind      string
	Condition LanguageCondition
	Children  []int
	Negate    bool
}

type ReadingRule struct {
	Surface, Reading string
	Context          []ReadingContext
}

// The original range syntax includes phrase-aware scopes. Keep it intact
// instead of treating all ranges as offsets into a flat list of words.
type ReadingContext struct {
	Direction  string
	Begin, End int
	All        bool
	Condition  int
}

// EvaluateCondition evaluates a word predicate, including compound predicates.
// The caller supplies boundary flags from its sentence/phrase structure; this
// method does not flatten the separate context ranges into word offsets.
func (r *ReadingRules) EvaluateCondition(id int, word LanguageWord, bos, eos bool) (bool, error) {
	var evaluate func(int, int) (bool, error)
	evaluate = func(id, depth int) (bool, error) {
		if depth > 1024 {
			return false, fmt.Errorf("reading predicate recursion exceeds limit")
		}
		p, ok := r.Conditions[id]
		if !ok {
			return false, fmt.Errorf("missing CD:%d", id)
		}
		var value bool
		var err error
		switch p.Kind {
		case "word":
			value, err = p.Condition.Evaluate(word)
		case "bos":
			value = bos
		case "eos":
			value = eos
		case "and":
			value = true
			for _, child := range p.Children {
				value, err = evaluate(child, depth+1)
				if err != nil || !value {
					break
				}
			}
		default:
			return false, fmt.Errorf("unsupported reading predicate %q", p.Kind)
		}
		if err != nil {
			return false, err
		}
		return value != p.Negate, nil
	}
	return evaluate(id, 0)
}

var readingScope = regexp.MustCompile(`^([PN])(?:\[(-?\d+):(-?\d+)\])?(A)?$`)

// ParseReadingRules consumes a UTF-8 conversion of the observed rule lines.
// The caller supplies the dependency graph when executing phrase conditions.
func ParseReadingRules(reader io.Reader) (*ReadingRules, error) {
	data, err := io.ReadAll(io.LimitReader(reader, 64*1024*1024+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 64*1024*1024 {
		return nil, fmt.Errorf("reading rule table exceeds 64 MiB")
	}
	result := &ReadingRules{Conditions: make(map[int]ReadingPredicate)}
	ss := make(map[int]string)
	ms := make(map[int][]int)
	mi := make(map[int][]uint32)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	var active *ReadingRule
	line := 0
	ref := func(token, prefix string) (int, error) {
		if !strings.HasPrefix(token, prefix+":") {
			return 0, fmt.Errorf("expected %s reference: %q", prefix, token)
		}
		id, err := strconv.Atoi(strings.TrimPrefix(token, prefix+":"))
		if err != nil || id <= 0 {
			return 0, fmt.Errorf("invalid table reference %q", token)
		}
		return id, nil
	}
	fail := func(err error) (*ReadingRules, error) { return nil, fmt.Errorf("reading rule line %d: %w", line, err) }
	for scanner.Scan() {
		line++
		raw := strings.TrimLeft(scanner.Text(), " \t")
		text := strings.TrimSpace(raw)
		if !utf8.ValidString(text) {
			return fail(fmt.Errorf("rule lines must be UTF-8"))
		}
		if text == "" || strings.HasPrefix(text, ";") {
			continue
		}
		parts := strings.Fields(text)
		if active != nil {
			if text == "}" {
				current := 0
				for _, c := range active.Context {
					if c.Direction == "C" {
						current++
					}
				}
				if current != 1 {
					return fail(fmt.Errorf("reading rule must have one current-word condition"))
				}
				result.Rules = append(result.Rules, *active)
				active = nil
				continue
			}
			if len(parts) != 2 {
				return fail(fmt.Errorf("invalid context condition"))
			}
			id, err := ref(parts[1], "CD")
			if err != nil {
				return fail(err)
			}
			c := ReadingContext{Direction: parts[0], Condition: id}
			if parts[0] != "C" {
				match := readingScope.FindStringSubmatch(parts[0])
				if match == nil {
					return fail(fmt.Errorf("unsupported context scope %q", parts[0]))
				}
				c.Direction, c.Begin, c.End, c.All = match[1], 1, 1, match[4] != ""
				if match[2] != "" {
					c.Begin, err = strconv.Atoi(match[2])
					if err != nil {
						return fail(err)
					}
					c.End, err = strconv.Atoi(match[3])
					if err != nil {
						return fail(err)
					}
				}
			}
			active.Context = append(active.Context, c)
			continue
		}
		switch {
		case text == "A 0":
		case parts[0] == "{":
			if len(parts) != 3 {
				return fail(fmt.Errorf("invalid reading rule header"))
			}
			first, err := ref(parts[1], "SS")
			if err != nil {
				return fail(err)
			}
			second, err := ref(parts[2], "SS")
			if err != nil {
				return fail(err)
			}
			surface, ok := ss[first]
			if !ok {
				return fail(fmt.Errorf("missing surface string %d", first))
			}
			reading, ok := ss[second]
			if !ok {
				return fail(fmt.Errorf("missing reading string %d", second))
			}
			active = &ReadingRule{Surface: surface, Reading: reading}
		case strings.HasPrefix(parts[0], "SS:"):
			separator := strings.IndexAny(raw, " \t")
			if separator < 0 {
				return fail(fmt.Errorf("invalid string table entry"))
			}
			id, err := ref(parts[0], "SS")
			if err != nil {
				return fail(err)
			}
			if _, exists := ss[id]; exists {
				return fail(fmt.Errorf("duplicate SS:%d", id))
			}
			// Native CP932 parsing treats the full-width space as a string
			// value. Unicode Fields/TrimSpace would silently discard it.
			ss[id] = strings.Trim(raw[separator+1:], " \t")
		case strings.HasPrefix(parts[0], "MS:"):
			id, err := ref(parts[0], "MS")
			if err != nil {
				return fail(err)
			}
			if _, exists := ms[id]; exists {
				return fail(fmt.Errorf("duplicate MS:%d", id))
			}
			ms[id] = []int{}
			for _, token := range parts[1:] {
				value, err := ref(token, "SS")
				if err != nil {
					return fail(err)
				}
				ms[id] = append(ms[id], value)
			}
		case strings.HasPrefix(parts[0], "MI:"):
			id, err := ref(parts[0], "MI")
			if err != nil {
				return fail(err)
			}
			if _, exists := mi[id]; exists {
				return fail(fmt.Errorf("duplicate MI:%d", id))
			}
			mi[id] = []uint32{}
			for _, token := range parts[1:] {
				value, err := strconv.ParseUint(token, 0, 32)
				if err != nil {
					return fail(err)
				}
				mi[id] = append(mi[id], uint32(value))
			}
		case strings.HasPrefix(parts[0], "CD:"):
			id, err := ref(parts[0], "CD")
			if err != nil {
				return fail(err)
			}
			if _, exists := result.Conditions[id]; exists {
				return fail(fmt.Errorf("duplicate CD:%d", id))
			}
			p := ReadingPredicate{Kind: "word"}
			args := parts[1:]
			if len(args) == 0 {
				return fail(fmt.Errorf("empty predicate"))
			}
			if args[0] == "!" {
				p.Negate = true
				args = args[1:]
			}
			if len(args) == 1 && (args[0] == "BOS" || args[0] == "EOS") {
				p.Kind = strings.ToLower(args[0])
			} else if len(args) >= 3 && args[0] == "&" {
				p.Kind = "and"
				for _, token := range args[1:] {
					child, err := ref(token, "CD")
					if err != nil {
						return fail(err)
					}
					p.Children = append(p.Children, child)
				}
			} else {
				if len(args) != 3 {
					return fail(fmt.Errorf("invalid predicate"))
				}
				operation := args[0]
				negate := strings.HasPrefix(operation, "!")
				operation = strings.TrimPrefix(operation, "!")
				integer := strings.HasSuffix(operation, "_I")
				operation = strings.TrimSuffix(operation, "_I")
				p.Condition = LanguageCondition{Field: args[1], Operator: operation, Negate: negate}
				if operation != "EQ" && operation != "IN" {
					return fail(fmt.Errorf("unsupported operator %q", operation))
				}
				if integer {
					if operation == "EQ" {
						value, err := strconv.ParseUint(args[2], 0, 32)
						if err != nil {
							return fail(err)
						}
						p.Condition.Integers = []uint32{uint32(value)}
					} else {
						set, err := ref(args[2], "MI")
						if err != nil {
							return fail(err)
						}
						values, ok := mi[set]
						if !ok {
							return fail(fmt.Errorf("missing MI:%d", set))
						}
						p.Condition.Integers = append([]uint32{}, values...)
					}
				} else {
					var ids []int
					if operation == "EQ" {
						value, err := ref(args[2], "SS")
						if err != nil {
							return fail(err)
						}
						ids = []int{value}
					} else {
						set, err := ref(args[2], "MS")
						if err != nil {
							return fail(err)
						}
						var ok bool
						ids, ok = ms[set]
						if !ok {
							return fail(fmt.Errorf("missing MS:%d", set))
						}
					}
					for _, value := range ids {
						text, ok := ss[value]
						if !ok {
							return fail(fmt.Errorf("missing SS:%d", value))
						}
						p.Condition.Strings = append(p.Condition.Strings, text)
					}
				}
				if _, err := p.Condition.Evaluate(LanguageWord{}); err != nil {
					return fail(err)
				}
			}
			result.Conditions[id] = p
		default:
			return fail(fmt.Errorf("unsupported rule line %q", text))
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if active != nil || len(result.Rules) == 0 {
		return nil, fmt.Errorf("incomplete reading rule table")
	}
	visiting, checked := make(map[int]bool), make(map[int]bool)
	var validate func(int) error
	validate = func(id int) error {
		if checked[id] {
			return nil
		}
		if visiting[id] {
			return fmt.Errorf("cyclic CD:%d", id)
		}
		p, ok := result.Conditions[id]
		if !ok {
			return fmt.Errorf("missing CD:%d", id)
		}
		visiting[id] = true
		for _, child := range p.Children {
			if err := validate(child); err != nil {
				return err
			}
		}
		delete(visiting, id)
		checked[id] = true
		return nil
	}
	for id := range result.Conditions {
		if err := validate(id); err != nil {
			return nil, err
		}
	}
	for _, rule := range result.Rules {
		for _, c := range rule.Context {
			if err := validate(c.Condition); err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}
