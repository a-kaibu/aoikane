package httpapi

import (
	"crypto/subtle"
	_ "embed"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/a-kaibu/aoikane/internal/audio"
	"github.com/a-kaibu/aoikane/internal/engine"
)

func Handler(s engine.Synthesizer, token string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		jsonReply(w, 200, map[string]any{"status": "ok", "engine": "A.I.VOICE2", "voices": len(s.Voices())})
	})
	mux.HandleFunc("GET /v1/voices", func(w http.ResponseWriter, r *http.Request) { jsonReply(w, 200, s.Voices()) })
	mux.HandleFunc("POST /v1/synthesize", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
			jsonReply(w, 415, map[string]string{"error": "Content-Type must be application/json"})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 128<<10)
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		var request engine.Request
		if err := dec.Decode(&request); err != nil {
			jsonReply(w, 400, map[string]string{"error": "invalid JSON request: " + err.Error()})
			return
		}
		var extra any
		if err := dec.Decode(&extra); err != io.EOF {
			jsonReply(w, 400, map[string]string{"error": "request must contain one JSON object"})
			return
		}
		request.Defaults()
		if err := request.Validate(); err != nil {
			jsonReply(w, 400, map[string]string{"error": err.Error()})
			return
		}
		if request.Voice != "" {
			found := false
			for _, v := range s.Voices() {
				if v.ID == request.Voice {
					found = true
				}
			}
			if !found {
				jsonReply(w, 400, map[string]string{"error": "unknown voice"})
				return
			}
		}
		wav, err := s.Synthesize(r.Context(), request)
		if err != nil {
			if r.Context().Err() != nil {
				return
			}
			jsonReply(w, 422, map[string]string{"error": err.Error()})
			return
		}
		data, err := audio.Convert(wav, request.Format)
		if err != nil {
			jsonReply(w, 422, map[string]string{"error": err.Error()})
			return
		}
		mime := "audio/wav"
		if request.Format == "mp3" {
			mime = "audio/mpeg"
		}
		w.Header().Set("Content-Type", mime)
		w.Header().Set("Content-Disposition", `attachment; filename="speech.`+request.Format+`"`)
		w.Write(data)
	})
	mux.HandleFunc("GET /openapi.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, OpenAPI)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if token != "" && subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+token)) != 1 {
			jsonReply(w, 401, map[string]string{"error": "unauthorized"})
			return
		}
		mux.ServeHTTP(w, r)
	})
}
func jsonReply(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}

//go:embed openapi.json
var OpenAPI string
