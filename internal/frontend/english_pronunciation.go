package frontend

import "fmt"

type EnglishPronunciationStep struct {
	Index    int
	Features EnglishDecisionFeatures
	Before   EnglishInitialCandidate
	Applied  EnglishInitialCandidate
	Final    EnglishInitialCandidate
	Results  [4]string
}

type EnglishPronunciationChunk struct {
	Prepared   []PreparedEnglishSegment
	Candidates []EnglishInitialCandidate
	Readings   []string
	Steps      []EnglishPronunciationStep // native reverse processing order
}

// ResolvePronunciationChunk reconstructs the normal-mode candidate pass in
// 1a6e00. Segment considers the first 30 ASCII bytes; callers must handle later
// chunks separately. Readings are internal phonetic strings, not kana.
func (r *EnglishSegmentRules) ResolvePronunciationChunk(text string, trees *EnglishDecisionTrees) (EnglishPronunciationChunk, error) {
	var out EnglishPronunciationChunk
	if trees == nil || len(trees.Trees) != 4 {
		return out, fmt.Errorf("missing English decision trees")
	}
	units, err := r.Segment(text)
	if err != nil {
		return out, err
	}
	out.Prepared, err = r.Prepare(units)
	if err != nil {
		return out, err
	}
	out.Candidates, err = r.InitialCandidates(units)
	if err != nil {
		return out, err
	}
	out.Readings = make([]string, len(units))
	out.Steps = make([]EnglishPronunciationStep, 0, len(units))
	accumulated, previousCost := 0, 0
	for i := len(units) - 1; i >= 0; i-- {
		features, err := BuildEnglishDecisionFeatures(out.Prepared, i, accumulated, previousCost)
		if err != nil {
			return out, err
		}
		results, err := trees.Evaluate(features.Strings[:], features.Integers[:])
		if err != nil {
			return out, err
		}
		before := out.Candidates[i]
		applied, err := ApplyEnglishRuleResults(before, results)
		if err != nil {
			return out, err
		}
		final := applied
		// The native post-rule correction precedes concatenation.
		if final.Primary == "p_o_s_u_t_o_" && out.Prepared[i].Primary == "tt" {
			final.Primary = "t_u_t_u_"
		}
		out.Candidates[i] = final
		reading := final.Primary + final.CombinedVowels + final.Tail
		if final.Cost != 0 {
			reading = "Q_" + reading
		}
		out.Readings[i] = reading
		out.Steps = append(out.Steps, EnglishPronunciationStep{Index: i, Features: features, Before: before, Applied: applied, Final: final, Results: results})
		previousCost = final.Cost
		accumulated += previousCost
	}
	return out, nil
}

// ResolveEnglishFallback reconstructs 1a6c60, including its 421-byte append
// limit. The native wrapper calls 1a6e00 once and ignores its remaining-input
// pointer: inputs of 30 or more letters use only the first 30, without '$'.
// English dictionary lookup preceding this fallback is a separate stage.
func (r *EnglishSegmentRules) ResolveEnglishFallback(text string, trees *EnglishDecisionTrees, kana *EnglishKanaRules, codec *CP932Codec) (string, error) {
	if codec == nil {
		return "", fmt.Errorf("missing CP932 codec")
	}
	if err := kana.Validate(); err != nil {
		return "", err
	}
	chunk, err := r.ResolvePronunciationChunk(text, trees)
	if err != nil {
		return "", err
	}
	var result []byte
	for _, reading := range chunk.Readings {
		result, err = kana.AppendCP932(result, reading, 421, codec)
		if err != nil {
			return "", err
		}
	}
	return codec.Decode(result)
}
