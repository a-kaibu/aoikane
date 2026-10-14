// Copyright (c) 2001-2008, Taku Kudo. See CABOCHA-LICENSE.
// Darts traversal: Copyright (c) 2008-2011, Susumu Yata. See DARTS-LICENSE.
package cabocha

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strings"
)

// SVM is the immutable degree-two FastSVMModel used by the dependency parser.
// Its dictionaries contain bytes in the model's charset (SHIFT_JIS in A.I.VOICE2).
// The model format and pair expansion follow CaboCha 96e100a62f6dadceb490cdd61c8c15fb561f6674.
type SVM struct {
	parameters                               map[string]string
	normalize                                float32
	bias                                     int32
	frequent                                 int
	dictionary, pairs, nodes, singles, dense []uint32
}

func LoadSVM(path string) (*SVM, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseSVM(b)
}

// ParseSVM copies the model arrays; subsequent changes to b cannot change scores.
func ParseSVM(b []byte) (*SVM, error) {
	fail := func() (*SVM, error) { return nil, fmt.Errorf("invalid CaboCha SVM model") }
	if len(b) < 36 {
		return fail()
	}
	u32 := func(o int) uint32 { return binary.LittleEndian.Uint32(b[o:]) }
	if uint64(u32(0)^0xef522177) != uint64(len(b)) || u32(4) != 102 {
		return fail()
	}
	paramSize := uint64(u32(8))
	if paramSize%8 != 0 || paramSize+36 > uint64(len(b)) {
		return fail()
	}
	end := 12 + int(paramSize)
	params := strings.Split(strings.TrimRight(string(b[12:end]), "\x00"), "\t")
	if len(params)%2 != 0 {
		return fail()
	}
	m := &SVM{parameters: make(map[string]string)}
	for i := 0; i < len(params); i += 2 {
		if params[i] == "" || strings.ContainsRune(params[i+1], 0) {
			return fail()
		}
		if _, ok := m.parameters[params[i]]; ok {
			return fail()
		}
		m.parameters[params[i]] = params[i+1]
	}
	if m.parameters["degree"] != "2" {
		return fail()
	}
	m.normalize = math.Float32frombits(u32(end))
	m.bias = int32(u32(end + 4))
	nf, freq := uint64(u32(end+8)), uint64(u32(end+12))
	dicBytes, pairBytes := uint64(u32(end+16)), uint64(u32(end+20))
	if nf == 0 || nf > math.MaxInt32 || freq == 0 || freq > nf || dicBytes == 0 || pairBytes == 0 || dicBytes%4 != 0 || pairBytes%4 != 0 || m.normalize <= 0 || math.IsInf(float64(m.normalize), 0) || math.IsNaN(float64(m.normalize)) {
		return fail()
	}
	if uint64(end)+24+dicBytes+pairBytes+nf*8+freq*(freq-1)*2 != uint64(len(b)) {
		return fail()
	}
	offset := end + 24
	read := func(count uint64) []uint32 {
		values := make([]uint32, int(count))
		for i := range values {
			values[i] = u32(offset)
			offset += 4
		}
		return values
	}
	m.frequent = int(freq)
	m.dictionary = read(dicBytes / 4)
	m.pairs = read(pairBytes / 4)
	m.nodes = read(nf)
	m.singles = read(nf)
	m.dense = read(freq * (freq - 1) / 2)
	for _, n := range m.nodes {
		if uint64(n) >= uint64(len(m.pairs)) {
			return fail()
		}
	}
	return m, nil
}

func (m *SVM) Parameter(key string) string { return m.parameters[key] }
func (m *SVM) FeatureCount() int           { return len(m.singles) }
func (m *SVM) FrequentFeatureCount() int   { return m.frequent }

// ID looks up an already encoded feature; an absent feature returns -1.
func (m *SVM) ID(key []byte) (int, error) {
	id, err := dartsMatch(m.dictionary, 0, key)
	if err == nil && id >= len(m.singles) {
		return -1, fmt.Errorf("SVM dictionary feature out of range")
	}
	return id, err
}

func dartsOffset(unit uint32) uint32 { return (unit >> 10) << ((unit & (1 << 9)) >> 6) }
func dartsMatch(array []uint32, node uint32, key []byte) (int, error) {
	for _, label := range key {
		if uint64(node) >= uint64(len(array)) {
			return -1, fmt.Errorf("SVM trie node out of range")
		}
		node ^= dartsOffset(array[node]) ^ uint32(label)
		if uint64(node) >= uint64(len(array)) {
			return -1, fmt.Errorf("SVM trie edge out of range")
		}
		if array[node]&0x800000ff != uint32(label) {
			return -1, nil
		}
	}
	if uint64(node) >= uint64(len(array)) {
		return -1, fmt.Errorf("SVM trie node out of range")
	}
	unit := array[node]
	if unit&256 == 0 {
		return -1, nil
	}
	leaf := node ^ dartsOffset(unit)
	if uint64(leaf) >= uint64(len(array)) {
		return -1, fmt.Errorf("SVM trie leaf out of range")
	}
	return int(array[leaf] & 0x7fffffff), nil
}

// svmKey reproduces CaboCha's incrementing BER encoding (not standard varint).
func svmKey(id int) []byte {
	value := uint32(id) + 1
	key := []byte{byte(value & 127)}
	for value >>= 7; value != 0; value >>= 7 {
		value++
		key[len(key)-1] |= 128
		key = append(key, byte(value&127))
	}
	return key
}

// Score expects the sorted, deduplicated IDs produced by the dependency parser.
// Like the original parser, each vector must contain a frequent feature.
func (m *SVM) Score(ids []int) (float64, error) {
	score, err := m.scoreSum(ids)
	if err != nil {
		return 0, err
	}
	// MSVC converts the int32 accumulator to float32, multiplies in float32,
	// then widens to the double return type. Keep both roundings explicit.
	return float64(float32(float32(score) * m.normalize)), nil
}

func (m *SVM) scoreSum(ids []int) (int32, error) {
	if len(ids) == 0 || ids[0] < 0 || ids[0] >= m.frequent {
		return 0, fmt.Errorf("SVM requires a frequent feature")
	}
	frequent := 0
	keys := make([][]byte, len(ids))
	for i, id := range ids {
		if id < 0 || id >= len(m.singles) || (i > 0 && id <= ids[i-1]) {
			return 0, fmt.Errorf("SVM feature IDs must be sorted, unique, and in range")
		}
		if id < m.frequent {
			frequent++
		}
		keys[i] = svmKey(id)
	}
	score := -m.bias
	for i, id := range ids {
		score += int32(m.singles[id])
		for j := i + 1; j < len(ids); j++ {
			if j < frequent {
				// Packed upper triangle, without the diagonal.
				offset := int(int64(id)*int64(2*m.frequent-3-id)/2 - 1 + int64(ids[j]))
				score += int32(m.dense[offset])
			} else if node := m.nodes[id]; node != 0 {
				value, err := dartsMatch(m.pairs, node, keys[j])
				if err != nil {
					return 0, err
				}
				if value >= 0 {
					score += int32(value) - 0xfffff
				}
			}
		}
	}
	return score, nil
}
