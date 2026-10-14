package frontend

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

type EnglishKanaRule struct {
	Code string `json:"code"`
	Kana string `json:"kana"`
}

type EnglishKanaRules struct {
	Source string            `json:"source_sha256"`
	Rules  []EnglishKanaRule `json:"rules"`
}

func OpenEnglishKanaRules(path string) (*EnglishKanaRules, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) > 128*1024 {
		return nil, fmt.Errorf("English kana rules exceed limit")
	}
	var r EnglishKanaRules
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return &r, nil
}

func (r *EnglishKanaRules) Validate() error {
	if r == nil || r.Source != "711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17" || len(r.Rules) != 148 {
		return fmt.Errorf("invalid English kana rules")
	}
	for _, rule := range r.Rules {
		if len(rule.Code) == 0 || len(rule.Code) > 32 || !utf8.ValidString(rule.Kana) || len(rule.Kana) == 0 || len(rule.Kana) > 64 {
			return fmt.Errorf("invalid English kana entry")
		}
		for _, c := range rule.Code {
			if c == 0 || c > 127 {
				return fmt.Errorf("non-ASCII English kana code")
			}
		}
		for _, c := range rule.Kana {
			if c == 0 {
				return fmt.Errorf("NUL in English kana entry")
			}
		}
	}
	return nil
}

// Convert reconstructs 1a5d60. Pairs beginning with '_' and an odd trailing
// byte are discarded before ordered prefix matching. Unmatched pairs are
// skipped; table order, including duplicate keys, is significant.
func (r *EnglishKanaRules) Convert(reading string) (string, error) {
	if err := r.Validate(); err != nil {
		return "", err
	}
	if len(reading) > 4096 {
		return "", fmt.Errorf("English phonetic input exceeds limit")
	}
	for _, c := range reading {
		if c == 0 || c > 127 {
			return "", fmt.Errorf("English phonetic input requires non-NUL ASCII")
		}
	}
	filtered := make([]byte, 0, len(reading))
	for i := 0; i+1 < len(reading); i += 2 {
		if reading[i] != '_' {
			filtered = append(filtered, reading[i], reading[i+1])
		}
	}
	input := string(filtered)
	var out strings.Builder
	for len(input) > 0 {
		matched := false
		for _, rule := range r.Rules {
			if strings.HasPrefix(input, rule.Code) {
				out.WriteString(rule.Kana)
				input = input[len(rule.Code):]
				matched = true
				break
			}
		}
		if !matched {
			if len(input) < 2 {
				break
			}
			input = input[2:]
		}
	}
	return out.String(), nil
}

// AppendCP932 reconstructs 1a6410's byte-buffer append. The native limit is
// checked before writing a pair, so an even limit can produce limit bytes
// followed by a NUL. The returned slice excludes that NUL. Only consecutive
// identical ッ or ー pairs are suppressed, including at the prefix boundary.
func (r *EnglishKanaRules) AppendCP932(prefix []byte, reading string, capacity int, codec *CP932Codec) ([]byte, error) {
	if codec == nil || capacity < 1 || capacity > 65536 || len(prefix) >= capacity {
		return nil, fmt.Errorf("invalid English kana append buffer")
	}
	kana, err := r.Convert(reading)
	if err != nil {
		return nil, err
	}
	encoded, err := codec.Encode(kana)
	if err != nil {
		return nil, err
	}
	if len(encoded)%2 != 0 {
		return nil, fmt.Errorf("English kana requires two-byte CP932 pairs")
	}
	result := append([]byte{}, prefix...)
	for i := 0; i+1 < len(encoded); i += 2 {
		if len(result) >= capacity-1 {
			continue
		}
		a, b := encoded[i], encoded[i+1]
		if (a == 0x83 && b == 0x62 || a == 0x81 && b == 0x5b) && len(result) > 1 && result[len(result)-2] == a && result[len(result)-1] == b {
			continue
		}
		result = append(result, a, b)
	}
	return result, nil
}
