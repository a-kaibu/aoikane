package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/a-kaibu/aoikane/internal/engine"
)

type Client struct {
	URL, Token string
	HTTP       *http.Client
}

func NewClient(address, token string) (*Client, error) {
	u, err := url.Parse(address)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("--url must be an http(s) URL")
	}
	return &Client{URL: strings.TrimRight(address, "/"), Token: token, HTTP: &http.Client{Timeout: 5 * time.Minute}}, nil
}
func (c *Client) request(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.URL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	limit := int64(256<<20) + 44
	if res.StatusCode != 200 {
		limit = 64 << 10
	}
	b, err := io.ReadAll(io.LimitReader(res.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("API response is too large")
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("API returned %s: %s", res.Status, strings.TrimSpace(string(b)))
	}
	return b, nil
}
func (c *Client) Synthesize(ctx context.Context, r engine.Request) ([]byte, error) {
	r.Defaults()
	if err := r.Validate(); err != nil {
		return nil, err
	}
	b, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	b, err = c.request(ctx, "POST", "/v1/synthesize", b)
	if err != nil {
		return nil, err
	}
	if r.Format == "mp3" {
		if len(b) < 4 || b[0] != 0xff || b[1]&0xe0 != 0xe0 {
			return nil, fmt.Errorf("API returned invalid MP3")
		}
		return b, nil
	}
	if len(b) < 44 || string(b[:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		return nil, fmt.Errorf("API returned invalid WAV")
	}
	return b, nil
}
func (c *Client) ListVoices(ctx context.Context) ([]engine.Voice, error) {
	b, e := c.request(ctx, "GET", "/v1/voices", nil)
	if e != nil {
		return nil, e
	}
	var voices []engine.Voice
	e = json.Unmarshal(b, &voices)
	return voices, e
}
