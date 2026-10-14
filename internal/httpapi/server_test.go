package httpapi

import (
	"bytes"
	"context"
	"net/http/httptest"
	"testing"

	"github.com/a-kaibu/aoikane/internal/engine"
)

type fakeEngine struct{}

func (fakeEngine) Voices() []engine.Voice {
	return []engine.Voice{{ID: "fixture", Name: "Fixture", SampleRate: 48000}}
}
func (fakeEngine) Synthesize(ctx context.Context, r engine.Request) ([]byte, error) {
	return engine.WAV(make([]byte, 4800), 48000)
}
func (fakeEngine) Close() error { return nil }
func TestAPIValidation(t *testing.T) {
	h := Handler(fakeEngine{}, "secret")
	for _, test := range []struct {
		body, token string
		status      int
	}{
		{`{"text":"hello"}`, "", 401}, {`{"text":"hello","format":"mp3"}`, "secret", 200}, {`{"text":""}`, "secret", 400}, {`{"text":"hello","voice":"unknown"}`, "secret", 400}, {`{"text":"hello","surprise":true}`, "secret", 400}, {`{"text":"hello"} {}`, "secret", 400}, {`{"text":"hello","format":"ogg"}`, "secret", 400},
	} {
		r := httptest.NewRequest("POST", "/v1/synthesize", bytes.NewBufferString(test.body))
		r.Header.Set("Content-Type", "application/json")
		if test.token != "" {
			r.Header.Set("Authorization", "Bearer "+test.token)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != test.status {
			t.Fatalf("%s: status %d, want %d: %s", test.body, w.Code, test.status, w.Body.String())
		}
		if w.Code == 200 && w.Header().Get("Content-Type") != "audio/mpeg" {
			t.Fatal("wrong MP3 content type")
		}
	}
}
func TestRemoteClient(t *testing.T) {
	s := httptest.NewServer(Handler(fakeEngine{}, "secret"))
	defer s.Close()
	c, err := NewClient(s.URL, "secret")
	if err != nil {
		t.Fatal(err)
	}
	voices, err := c.ListVoices(context.Background())
	if err != nil || len(voices) != 1 {
		t.Fatalf("voices: %v %v", voices, err)
	}
	for _, format := range []string{"wav", "mp3"} {
		r := engine.Request{Text: "hello", Format: format}
		r.Defaults()
		b, err := c.Synthesize(context.Background(), r)
		if err != nil || len(b) == 0 {
			t.Fatalf("%s: %v", format, err)
		}
	}
	c.Token = "incorrect"
	if _, err = c.ListVoices(context.Background()); err == nil {
		t.Fatal("bad token accepted")
	}
}

