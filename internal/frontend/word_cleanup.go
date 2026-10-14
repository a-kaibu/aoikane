package frontend

import "fmt"

type CleanupWord struct {
	KanaMergeWord
	HasChildren bool `json:"has_children"`
}

type WordCleanupResult struct {
	Current      CleanupWord
	Next         CleanupWord
	Keep         bool
	ResetReading bool
}

// CleanupGroupWord keeps the selected pronunciation's accent attached to its
// word while the cleanup pass changes POS values and removes separators.
// nil means there is no selected accent; a pointer to "" is a selected empty
// accent. These states must not be inferred from the pronunciation spelling.
type CleanupGroupWord struct {
	CleanupWord
	SelectedAccent *string `json:"selected_accent"`
}

// CleanupWordGroup reconstructs the left-to-right cleanup loop at 1a494c.
// Boundary words are explicit: native groups can border container nodes, and
// replacing those nodes with empty leaf words would change the bypass rule.
// The input slice is preserved. Mutations to the next word become visible to
// the following iteration, including when the current word is removed.
func CleanupWordGroup(words []CleanupGroupWord, previous, next CleanupGroupWord) ([]CleanupGroupWord, error) {
	out := make([]CleanupGroupWord, 0, len(words))
	remaining := append([]CleanupGroupWord(nil), words...)
	for i := range remaining {
		left := previous
		if len(out) != 0 {
			left = out[len(out)-1]
		}
		right := next
		if i+1 < len(remaining) {
			right = remaining[i+1]
		}
		current := remaining[i]
		var moras *int
		// Only this branch consumes the next accent in the native function.
		// Unrelated words must not fail because their accent is unparsed.
		if !current.HasChildren && !right.HasChildren && current.Surface == "・" && cleanupAlphabet(left.Surface, false) && cleanupAlphabet(right.Surface, false) && left.POS == 0x01010102 && right.POS == 0x01010101 && right.SelectedAccent != nil {
			record, err := ParseAccentText(*right.SelectedAccent, 0)
			if err != nil {
				return nil, fmt.Errorf("word %d selected accent: %w", i, err)
			}
			value := int(record.Second)
			moras = &value
		}
		result, err := CleanupWordNeighbors(left.CleanupWord, current.CleanupWord, right.CleanupWord, moras)
		if err != nil {
			return nil, fmt.Errorf("word %d: %w", i, err)
		}
		current.CleanupWord = result.Current
		if result.ResetReading {
			// 178350 replaces the reading record; 159a50 formats the
			// empty pronunciation's zero mora count as "*".
			accent := "*"
			current.SelectedAccent = &accent
		}
		right.CleanupWord = result.Next
		if i+1 < len(remaining) {
			remaining[i+1] = right
		} else if result.Next != next.CleanupWord {
			return nil, fmt.Errorf("word cleanup would mutate an external boundary word")
		}
		if result.Keep {
			out = append(out, current)
		}
	}
	return out, nil
}

func cleanupAlphabet(s string, alphabetic bool) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if alphabetic {
			if !(c >= 'Ａ' && c <= 'Ｚ' || c >= 'ａ' && c <= 'ｚ') {
				return false
			}
		} else if !(c >= 0x30a1 && c <= 0x30f6 || c == 0x30fc) {
			return false
		}
	}
	return true
}

// CleanupWordNeighbors reconstructs 1a5a90. Container nodes bypass cleanup.
// The special POS transition uses the next reading's parsed accent mora count,
// not a count inferred from its surface. The caller removes Current if !Keep.
func CleanupWordNeighbors(previous, current, next CleanupWord, nextAccentMoras *int) (WordCleanupResult, error) {
	out := WordCleanupResult{Current: current, Next: next, Keep: true}
	if current.HasChildren || next.HasChildren {
		return out, nil
	}
	clear := func() { out.Current.Pronunciation = ""; out.Current.Generated = nil; out.ResetReading = true }
	if current.Surface == "・" && cleanupAlphabet(previous.Surface, false) && cleanupAlphabet(next.Surface, false) {
		clear()
		if previous.POS == 0x01010102 && next.POS == 0x01010101 {
			if nextAccentMoras == nil {
				return out, fmt.Errorf("word cleanup requires selected accent mora count")
			}
			if *nextAccentMoras > 0 {
				out.Next.POS = 0x01090000
				out.Keep = false
				return out, nil
			}
		}
	}
	if current.Surface == "　" && cleanupAlphabet(previous.Surface, true) && cleanupAlphabet(next.Surface, true) {
		clear()
	}
	return out, nil
}
