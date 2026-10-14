package frontend

import (
	"fmt"
	"strings"
)

// ExpansionWord represents the fields touched by the word-chain expansion at
// 1903a0. ExpansionFirst/Last are the detached chain at native offsets 28/30,
// not the hierarchical child pointer at offset 20. Other word data remains
// attached to each node when the links are spliced.
type ExpansionWord struct {
	Previous, Next                *ExpansionWord
	ExpansionFirst, ExpansionLast *ExpansionWord
	Surface                       string
	Category                      int32
	Annotation                    string
}

// ExpandWordChain inserts a detached chain immediately after its owner and
// returns the last inserted word (the owner if there is no chain). The owner's
// category and annotation are inherited by the last word; earlier categories
// become 21. Annotations on earlier words remain unchanged.
// All surfaces are gathered on the owner, leaving inserted surfaces empty.
// Invalid links are rejected before changing any node.
func ExpandWordChain(owner *ExpansionWord) (*ExpansionWord, error) {
	if owner == nil {
		return nil, fmt.Errorf("missing expansion owner")
	}
	if owner.ExpansionFirst == nil {
		return owner, nil
	}
	if owner.ExpansionLast == nil || owner.Next == nil {
		return nil, fmt.Errorf("expansion requires a last word and following boundary")
	}
	chain := []*ExpansionWord{owner}
	seen := map[*ExpansionWord]bool{owner: true, owner.Next: true}
	for node := owner.ExpansionFirst; ; node = node.Next {
		if node == nil || seen[node] || len(chain) >= 65536 {
			return nil, fmt.Errorf("invalid detached expansion chain")
		}
		seen[node] = true
		chain = append(chain, node)
		if node == owner.ExpansionLast {
			break
		}
	}
	var surface strings.Builder
	for _, node := range chain {
		surface.WriteString(node.Surface)
	}
	last, following := owner.ExpansionLast, owner.Next
	category, annotation := owner.Category, owner.Annotation
	owner.Next = owner.ExpansionFirst
	owner.Next.Previous = owner
	last.Next = following
	following.Previous = last
	owner.ExpansionFirst, owner.ExpansionLast = nil, nil
	for _, node := range chain {
		node.Category = 21
		node.Surface = ""
	}
	last.Category, last.Annotation = category, annotation
	owner.Surface = surface.String()
	return last, nil
}
