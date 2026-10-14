package numeral

import "testing"

func TestIntegerFragmentObservedSound(t *testing.T) {
	// Native 10b010 returns observed without mutation; full formats also
	// match separate ordinary SDK captures. See integer-fragment-original.json.
	r := openRules(t, "bancharcx-kansai.txt")
	for _, tc := range []struct{ number, sound string }{
		{"十十", "ジュージュー"},
		{"千千", "センセン"},
		{"１百", "イチヒャク"},
		{"一百", "イチヒャク"},
		{"０億", "ゼロオク"},
	} {
		t.Run(tc.number, func(t *testing.T) {
			got, err := r.integerFragmentSound(tc.number)
			if err != nil || got != tc.sound {
				t.Fatalf("native sound %q: got %q, %v; want %q", tc.number, got, err, tc.sound)
			}
		})
	}
}
