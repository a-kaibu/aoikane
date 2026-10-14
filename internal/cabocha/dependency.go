// Copyright (c) 2001-2008, Taku Kudo. See CABOCHA-LICENSE.
package cabocha

import (
	"fmt"
	"slices"
)

type FeatureEncoder interface{ Encode(string) ([]byte, error) }
type Dependency struct {
	Link  int
	Score float32
}

// Candidate records one SVM decision, excluding links forced at sentence end.
type Candidate struct {
	Src, Dst int
	IDs      []int
	Score    float64
}

// Dependencies applies CaboCha's shift/reduce parser to Selector features.
// Encoding uses the same charset as the model; selection strings are UTF-8.
func (m *SVM) Dependencies(chunks []Selection, encoder FeatureEncoder) ([]Dependency, error) {
	return m.dependencies(chunks, encoder, nil)
}

func (m *SVM) dependencies(chunks []Selection, encoder FeatureEncoder, observe func(Candidate)) ([]Dependency, error) {
	result := make([]Dependency, len(chunks))
	for i := range result {
		result[i].Link = -1
	}
	if len(chunks) < 2 {
		return result, nil
	}
	if encoder == nil {
		return nil, fmt.Errorf("CaboCha dependency feature encoder is required")
	}
	// The original rewrites F into S/D and A into a/A. Keep source features
	// immutable and construct each candidate's keys without changing the tree.
	type features struct{ static, left, right, gap, child []string }
	grouped := make([]features, len(chunks))
	for i, chunk := range chunks {
		for _, feature := range chunk.Features {
			if feature == "" {
				continue
			}
			switch feature[0] {
			case 'F':
				grouped[i].static = append(grouped[i].static, feature)
			case 'L':
				grouped[i].left = append(grouped[i].left, feature)
			case 'R':
				grouped[i].right = append(grouped[i].right, feature)
			case 'G':
				grouped[i].gap = append(grouped[i].gap, feature)
			case 'A':
				grouped[i].child = append(grouped[i].child, feature)
			default:
				continue // CaboCha ignores features with other initial bytes.
			}
		}
	}
	children := make([][]int, len(chunks))
	lookupCache := make(map[string]int)
	estimate := func(src, dst int) (float64, error) {
		var ids []int
		add := func(key string) error {
			id, ok := lookupCache[key]
			if !ok {
				encoded, err := encoder.Encode(key)
				if err != nil {
					return err
				}
				id, err = m.ID(encoded)
				if err != nil {
					return err
				}
				lookupCache[key] = id
			}
			if id >= 0 {
				ids = append(ids, id)
			}
			return nil
		}
		addAll := func(list []string, prefix byte) error {
			for _, key := range list {
				if prefix != 0 {
					key = string(prefix) + key[1:]
				}
				if err := add(key); err != nil {
					return err
				}
			}
			return nil
		}
		distance := "DIST:6-"
		if dst-src == 1 {
			distance = "DIST:1"
		} else if dst-src <= 5 {
			distance = "DIST:2-5"
		}
		if err := add(distance); err != nil {
			return 0, err
		}
		if err := addAll(grouped[src].static, 'S'); err != nil {
			return 0, err
		}
		if err := addAll(grouped[dst].static, 'D'); err != nil {
			return 0, err
		}
		if src > 0 {
			if err := addAll(grouped[src-1].left, 0); err != nil {
				return 0, err
			}
		}
		if dst+1 < len(chunks) {
			if err := addAll(grouped[dst+1].right, 0); err != nil {
				return 0, err
			}
		}
		for _, child := range children[src] {
			if err := addAll(grouped[child].child, 'a'); err != nil {
				return 0, err
			}
		}
		for _, child := range children[dst] {
			if err := addAll(grouped[child].child, 'A'); err != nil {
				return 0, err
			}
		}
		brackets := 0
		for k := src + 1; k < dst; k++ {
			for _, key := range grouped[k].gap {
				switch key {
				case "GOB:1":
					brackets |= 1
				case "GCB:1":
					brackets |= 2
				default:
					if err := add(key); err != nil {
						return 0, err
					}
				}
			}
		}
		if err := add([]string{"GNB:1", "GOB:1", "GCB:1", "GBB:1"}[brackets]); err != nil {
			return 0, err
		}
		slices.Sort(ids)
		ids = slices.Compact(ids)
		score, err := m.Score(ids)
		if err == nil && observe != nil {
			observe(Candidate{src, dst, ids, score})
		}
		return score, err
	}
	agenda := []int{0}
	score := float64(0)
	for dst := 1; dst < len(chunks); dst++ {
		for len(agenda) > 0 {
			src := agenda[len(agenda)-1]
			if dst != len(chunks)-1 {
				var err error
				score, err = estimate(src, dst)
				if err != nil {
					return nil, err
				}
				if score <= 0 {
					break
				}
			}
			agenda = agenda[:len(agenda)-1]
			result[src] = Dependency{dst, float32(score)}
			children[dst] = append(children[dst], src)
		}
		agenda = append(agenda, dst)
	}
	return result, nil
}
