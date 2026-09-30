package voicekit

import (
	"context"
	"io"
	"net/http"
	"net/url"
)

// newRequest builds an authenticated request. It is shared by the buffered
// transport (send) and the streaming transport.
func (c *Client) newRequest(ctx context.Context, method, path string, query url.Values, contentType string, body io.Reader) (*http.Request, error) {
	endpoint := c.baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "voicekit-go/"+Version)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return req, nil
}

func setStr(body map[string]any, key, value string) {
	if value != "" {
		body[key] = value
	}
}

func setBool(body map[string]any, key string, value *bool) {
	if value != nil {
		body[key] = *value
	}
}

func setInt(body map[string]any, key string, value int) {
	if value > 0 {
		body[key] = value
	}
}

func setFloat(body map[string]any, key string, value float64) {
	if value != 0 {
		body[key] = value
	}
}

// queryParams renders non-empty string params, skipping blanks.
func queryParams(values map[string]string) url.Values {
	out := url.Values{}
	for key, value := range values {
		if value != "" {
			out.Set(key, value)
		}
	}
	return out
}
