// Package cabocha reconstructs the CaboCha steps the engine runs on its word
// graph: the CRF chunker (bunsetsu boundaries), IPA feature selection,
// and the SVM dependency parser.
package cabocha

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

// CRF is a CRF++ model (version 100) loaded for decoding.
type CRF struct {
	Labels    []string
	templates []template
	bigram    bool
	base      []int32
	check     []uint32
	alpha     []float32
	cost      float64
	xsize     int
}

type template struct {
	name  string // "U00", ...
	parts []tpart
}

// tpart is a literal or a %x[row,col] reference.
type tpart struct {
	literal  string
	row, col int
	ref      bool
}

// LoadCRF reads a model file.
func LoadCRF(path string) (*CRF, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseCRF(b)
}

// ParseCRF parses the binary model.
func ParseCRF(b []byte) (*CRF, error) {
	if len(b) < 32 {
		return nil, fmt.Errorf("short CRF model")
	}
	version := binary.LittleEndian.Uint32(b[0:])
	if version != 100 || binary.LittleEndian.Uint32(b[4:]) != 0 {
		return nil, fmt.Errorf("unsupported CRF++ model (version %d)", version)
	}
	m := &CRF{cost: math.Float64frombits(binary.LittleEndian.Uint64(b[8:]))}
	maxid := int(binary.LittleEndian.Uint32(b[16:]))
	m.xsize = int(binary.LittleEndian.Uint32(b[20:]))
	dsize := int(binary.LittleEndian.Uint32(b[24:]))
	ysize := int(binary.LittleEndian.Uint32(b[28:]))
	o := 32
	if o+ysize > len(b) {
		return nil, fmt.Errorf("truncated labels")
	}
	for _, l := range strings.Split(strings.TrimRight(string(b[o:o+ysize]), "\x00"), "\x00") {
		m.Labels = append(m.Labels, l)
	}
	o += ysize
	tsize := int(binary.LittleEndian.Uint32(b[o:]))
	o += 4
	if o+tsize > len(b) {
		return nil, fmt.Errorf("truncated templates")
	}
	for _, t := range strings.Split(strings.TrimRight(string(b[o:o+tsize]), "\x00"), "\x00") {
		if t == "" {
			continue
		}
		tp, bigram, err := parseTemplate(t)
		if err != nil {
			return nil, err
		}
		if bigram {
			m.bigram = true
			continue
		}
		m.templates = append(m.templates, tp)
	}
	o += tsize
	if dsize%8 != 0 || o+dsize+4*maxid > len(b) {
		return nil, fmt.Errorf("invalid CRF++ model sizes")
	}
	n := dsize / 8
	m.base = make([]int32, n)
	m.check = make([]uint32, n)
	for i := 0; i < n; i++ {
		m.base[i] = int32(binary.LittleEndian.Uint32(b[o+8*i:]))
		m.check[i] = binary.LittleEndian.Uint32(b[o+8*i+4:])
	}
	o += dsize
	m.alpha = make([]float32, maxid)
	for i := range m.alpha {
		m.alpha[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[o+4*i:]))
	}
	return m, nil
}

func parseTemplate(s string) (template, bool, error) {
	if s == "B" {
		return template{}, true, nil
	}
	i := strings.IndexByte(s, ':')
	if i < 0 {
		return template{}, false, fmt.Errorf("bad template %q", s)
	}
	t := template{name: s[:i]}
	rest := s[i+1:]
	for rest != "" {
		j := strings.Index(rest, "%x[")
		if j < 0 {
			t.parts = append(t.parts, tpart{literal: rest})
			break
		}
		if j > 0 {
			t.parts = append(t.parts, tpart{literal: rest[:j]})
		}
		k := strings.IndexByte(rest[j:], ']')
		if k < 0 {
			return t, false, fmt.Errorf("bad template %q", s)
		}
		fields := strings.Split(rest[j+3:j+k], ",")
		if len(fields) != 2 {
			return t, false, fmt.Errorf("bad template %q", s)
		}
		row, err1 := strconv.Atoi(fields[0])
		col, err2 := strconv.Atoi(fields[1])
		if err1 != nil || err2 != nil {
			return t, false, fmt.Errorf("bad template %q", s)
		}
		t.parts = append(t.parts, tpart{ref: true, row: row, col: col})
		rest = rest[j+k+1:]
	}
	return t, false, nil
}

// lookup finds a feature id in the double array (Darts exact match).
func (m *CRF) lookup(key string) int {
	b := int64(m.base[0])
	for i := 0; i < len(key); i++ {
		p := b + int64(key[i]) + 1
		if p < 0 || p >= int64(len(m.check)) || int64(m.check[p]) != b {
			return -1
		}
		b = int64(m.base[p])
	}
	if b < 0 || b >= int64(len(m.check)) {
		return -1
	}
	if int64(m.check[b]) == b && m.base[b] < 0 {
		return int(-m.base[b] - 1)
	}
	return -1
}

func expand(t template, tokens [][]string, pos int) (string, bool) {
	var sb strings.Builder
	sb.WriteString(t.name)
	sb.WriteByte(':')
	for _, p := range t.parts {
		if !p.ref {
			sb.WriteString(p.literal)
			continue
		}
		idx := pos + p.row
		switch {
		case idx < 0:
			sb.WriteString("_B-" + strconv.Itoa(-idx))
		case idx >= len(tokens):
			sb.WriteString("_B+" + strconv.Itoa(idx-len(tokens)+1))
		default:
			if p.col >= len(tokens[idx]) {
				return "", false
			}
			sb.WriteString(tokens[idx][p.col])
		}
	}
	return sb.String(), true
}

// Tag labels the tokens (each a row of columns) with the best label path.
func (m *CRF) Tag(tokens [][]string) []string {
	n, ny := len(tokens), len(m.Labels)
	if n == 0 {
		return nil
	}
	node := make([][]float64, n)
	for i := range node {
		node[i] = make([]float64, ny)
		for _, t := range m.templates {
			key, ok := expand(t, tokens, i)
			if !ok {
				continue
			}
			if id := m.lookup(key); id >= 0 {
				for y := 0; y < ny; y++ {
					node[i][y] += float64(m.alpha[id+y])
				}
			}
		}
		for y := range node[i] {
			node[i][y] *= m.cost
		}
	}
	trans := make([][]float64, ny)
	for y := range trans {
		trans[y] = make([]float64, ny)
	}
	if m.bigram {
		if id := m.lookup("B"); id >= 0 {
			for a := 0; a < ny; a++ {
				for b := 0; b < ny; b++ {
					trans[a][b] = float64(m.alpha[id+a*ny+b]) * m.cost
				}
			}
		}
	}
	score := make([][]float64, n)
	back := make([][]int, n)
	for i := range score {
		score[i] = make([]float64, ny)
		back[i] = make([]int, ny)
	}
	copy(score[0], node[0])
	for i := 1; i < n; i++ {
		for y := 0; y < ny; y++ {
			best, arg := math.Inf(-1), 0
			for a := 0; a < ny; a++ {
				if v := score[i-1][a] + trans[a][y]; v > best {
					best, arg = v, a
				}
			}
			score[i][y] = best + node[i][y]
			back[i][y] = arg
		}
	}
	y := 0
	for k := 1; k < ny; k++ {
		if score[n-1][k] > score[n-1][y] {
			y = k
		}
	}
	out := make([]string, n)
	for i := n - 1; i >= 0; i-- {
		out[i] = m.Labels[y]
		y = back[i][y]
	}
	return out
}

// Debug returns the node scores and the transition matrix (for tests).
func (m *CRF) Debug(tokens [][]string) ([][]float64, [][]float64) {
	n, ny := len(tokens), len(m.Labels)
	node := make([][]float64, n)
	for i := range node {
		node[i] = make([]float64, ny)
		for _, t := range m.templates {
			key, ok := expand(t, tokens, i)
			if !ok {
				continue
			}
			if id := m.lookup(key); id >= 0 {
				for y := 0; y < ny; y++ {
					node[i][y] += float64(m.alpha[id+y])
				}
			}
		}
	}
	trans := make([][]float64, ny)
	for y := range trans {
		trans[y] = make([]float64, ny)
	}
	if id := m.lookup("B"); id >= 0 {
		for a := 0; a < ny; a++ {
			for b := 0; b < ny; b++ {
				trans[a][b] = float64(m.alpha[id+a*ny+b])
			}
		}
	}
	return node, trans
}
