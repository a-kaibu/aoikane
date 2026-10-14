package frontend

import (
	"encoding/json"
	"fmt"
	"os"
)

type EnglishDecisionNode struct {
	Yes     int    `json:"yes"`
	No      int    `json:"no"`
	Feature int    `json:"feature"`
	Kind    int    `json:"kind"`
	Integer int32  `json:"integer"`
	Text    string `json:"text"`
}

type EnglishDecisionTree struct {
	Nodes  []EnglishDecisionNode `json:"nodes"`
	Leaves []string              `json:"leaves"`
}

type EnglishDecisionTrees struct {
	Source string                `json:"source_sha256"`
	Trees  []EnglishDecisionTree `json:"trees"`
}

func OpenEnglishDecisionTrees(path string) (*EnglishDecisionTrees, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) > 16*1024*1024 {
		return nil, fmt.Errorf("English decision tables exceed limit")
	}
	var r EnglishDecisionTrees
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	if r.Source != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(r.Trees) != 4 {
		return nil, fmt.Errorf("invalid English decision tables")
	}
	for _, t := range r.Trees {
		if err := t.Validate(); err != nil {
			return nil, err
		}
	}
	return &r, nil
}

func (t *EnglishDecisionTree) Validate() error {
	if t == nil || len(t.Nodes) > 32768 || len(t.Leaves) == 0 || len(t.Leaves) > 32768 {
		return fmt.Errorf("invalid English decision tree size")
	}
	for _, node := range t.Nodes {
		if node.Kind < -128 || node.Kind > 127 || node.Feature < 0 || node.Feature > 255 {
			return fmt.Errorf("invalid English decision predicate")
		}
		for _, branch := range [2]int{node.Yes, node.No} {
			if branch < 0 && -branch >= len(t.Nodes) || branch >= 0 && branch >= len(t.Leaves) {
				return fmt.Errorf("invalid English decision branch")
			}
		}
	}
	return nil
}

// Evaluate reconstructs 1a3d50. Negative destinations select a node;
// nonnegative destinations select a leaf. The native iteration bound is the
// node count and falls back to leaf zero if traversal exhausts that bound.
func (t *EnglishDecisionTree) Evaluate(strings []string, integers []int32) (string, error) {
	if t == nil || len(t.Leaves) == 0 {
		return "", fmt.Errorf("missing English decision tree")
	}
	index := 0
	for step := 0; step < len(t.Nodes); step++ {
		if index < 0 || index >= len(t.Nodes) {
			return "", fmt.Errorf("English decision node outside tree")
		}
		n := t.Nodes[index]
		matched := false
		if n.Kind > 0 {
			if n.Feature < 0 || n.Feature >= len(strings) {
				return "", fmt.Errorf("missing English string feature")
			}
			matched = strings[n.Feature] == n.Text
		} else {
			if n.Feature < 0 || n.Feature >= len(integers) {
				return "", fmt.Errorf("missing English integer feature")
			}
			if n.Kind == 0 {
				matched = integers[n.Feature] < n.Integer
			} else {
				matched = integers[n.Feature] == n.Integer
			}
		}
		branch := n.No
		if matched {
			branch = n.Yes
		}
		if branch >= 0 {
			if branch >= len(t.Leaves) {
				return "", fmt.Errorf("English decision leaf outside tree")
			}
			return t.Leaves[branch], nil
		}
		index = -branch
	}
	return t.Leaves[0], nil
}

func (r *EnglishDecisionTrees) Evaluate(strings []string, integers []int32) ([4]string, error) {
	var result [4]string
	if r == nil || len(r.Trees) != 4 {
		return result, fmt.Errorf("missing English decision tables")
	}
	for i := range result {
		value, err := r.Trees[i].Evaluate(strings, integers)
		if err != nil {
			return result, err
		}
		result[i] = value
	}
	return result, nil
}
