package frontend

import "fmt"

// BuildReadingContext reconstructs 191640's document context buffer. Words
// contain the CP932 surface bytes before the later word-processing passes;
// sourceBytes is the original input buffer's byte length, before normalization.
// The native builder reserves one byte within that length, appends whole words
// only, and stops at the first word that does not fit. The returned slice omits
// the native trailing NUL. Encoding and text normalization happen upstream.
func BuildReadingContext(words [][]byte, sourceBytes int) ([]byte, error) {
	if sourceBytes < 0 || sourceBytes > 1024*1024 || len(words) > 8192 {
		return nil, fmt.Errorf("invalid reading context source size")
	}
	// The original returns without allocating or changing an existing buffer
	// when its source size is zero. This constructor has no previous buffer.
	if sourceBytes == 0 {
		return nil, nil
	}
	result := make([]byte, 0, sourceBytes)
	for _, word := range words {
		if len(word) > sourceBytes-1-len(result) {
			break
		}
		result = append(result, word...)
	}
	return result, nil
}
