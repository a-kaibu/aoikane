package phrase

// ApplyKansaiBoundRules reconstructs the Kansai rule pass at RVA 0x135e80.
// Chunk-derived boundary prediction (137520) must run before this pass. The
// existing reading cache is preserved; this pass only changes boundary fields.
func (d *Document) ApplyKansaiBoundRules(rules *BoundRules) {
	var words []*Morph
	for _, m := range d.Morphs {
		if m == d.EOS {
			break
		}
		words = append(words, m)
	}
	for i, m := range words {
		kind, label := rules.Match(words, i)
		switch {
		case kind == 0:
			continue
		case kind >= 0x1f:
			m.Kind2 = kind
		case kind == 18:
			m.Kind = 17
		case m.Kind == 3 || m.Kind == 4 || m.Kind == 9:
			continue
		default:
			m.Kind = kind
		}
		m.Annotation = label
	}
}
