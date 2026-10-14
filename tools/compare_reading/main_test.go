package main

import (
	"encoding/json"
	"testing"
)

// Failed original processes and empty formatter captures must never produce a
// successful fidelity audit, even when the JSON contains a complete flag.
func TestRejectFailedOriginalCapture(t *testing.T) {
	const raw = `{"dll_sha256":"711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17","voice":"KotonohaAkane_ns_48","complete":true,"timed_out":false,"errors":[],"detach_reason":"process-terminated","crash":null,"corpus_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","native_report_sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","cases":[{"index":0,"text":"あ。","status":0,"readings":["<S>ア<F>"],"cp932_hex":["3c533e83413c463e"],"wav_bytes":100,"wav_sha256":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"}]}`
	fresh := func() capture {
		var c capture
		if err := json.Unmarshal([]byte(raw), &c); err != nil {
			t.Fatal(err)
		}
		return c
	}
	if err := fresh().validate(); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name         string
		breakCapture func(*capture)
	}{
		{"timeout", func(c *capture) { c.TimedOut = true }},
		{"partial", func(c *capture) { c.Complete = false }},
		{"crash", func(c *capture) { c.Crash = json.RawMessage(`{"summary":"crashed"}`) }},
		{"missing-crash-status", func(c *capture) { c.Crash = nil }},
		{"detached-live", func(c *capture) { c.Detach = "application-requested" }},
		{"no-formatter", func(c *capture) { c.Cases[0].Readings = nil; c.Cases[0].CP932 = nil }},
		{"failed-talk", func(c *capture) { c.Cases[0].Status = -1 }},
		{"missing-wave", func(c *capture) { c.Cases[0].WAVBytes = 0 }},
		{"partial-wave-report", func(c *capture) { c.NativeReportSHA = "" }},
		{"wrong-order", func(c *capture) { c.Cases[0].Index = 1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := fresh()
			tc.breakCapture(&c)
			if c.validate() == nil {
				t.Fatal("failed original capture accepted")
			}
		})
	}
}
