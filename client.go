// Package voicekit is the official Go SDK for the VoiceKit API — Russian
// text-to-speech, speech-to-text with diarization, voice cloning, voice
// biometrics, audio/video effects, call QA, semantic search and batch jobs.
//
//	client, err := voicekit.NewClient("YOUR_KEY")
//	if err != nil {
//		log.Fatal(err)
//	}
//	audio, err := client.Synthesize(ctx, "Привет! Это синтез русской речи.", &voicekit.SynthesizeOptions{
//		Voice:  "preset_anna",
//		Format: "mp3",
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//	os.WriteFile("speech.mp3", audio, 0o644)
//
// A Client is safe for concurrent use by multiple goroutines; create one and
// reuse it. Every call takes a context.Context and honours its deadline.
package voicekit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the public VoiceKit endpoint.
	DefaultBaseURL = "https://ttsapi.ru"

	// Version is the SDK release version.
	Version = "0.4.0"

	defaultTimeout = 120 * time.Second
)

// Client is a typed wrapper over the VoiceKit REST API.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

type config struct {
	baseURL string
	http    *http.Client
	timeout time.Duration
}

// Option customises a Client created by NewClient.
type Option func(*config)

// WithBaseURL overrides the API endpoint (default DefaultBaseURL).
func WithBaseURL(baseURL string) Option {
	return func(c *config) { c.baseURL = strings.TrimRight(baseURL, "/") }
}

// WithHTTPClient injects a custom *http.Client. The client's own Timeout wins
// over WithTimeout.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *config) { c.http = hc }
}

// WithTimeout sets the per-request timeout (default 120s).
func WithTimeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

// NewClient creates a client for the given API key. The key is required.
func NewClient(apiKey string, opts ...Option) (*Client, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("voicekit: apiKey is required")
	}
	cfg := config{baseURL: DefaultBaseURL, timeout: defaultTimeout}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.http == nil {
		cfg.http = &http.Client{Timeout: cfg.timeout}
	}
	return &Client{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(cfg.baseURL, "/"),
		http:    cfg.http,
	}, nil
}

// BaseURL returns the configured API endpoint.
func (c *Client) BaseURL() string { return c.baseURL }

// ────────────────────────── file attachments ──────────────────────────────

// Attachment is a file uploaded with a multipart/form-data request.
type Attachment struct {
	// Name is the file name reported to the server (e.g. "audio.wav").
	Name string
	// ContentType is the MIME type (e.g. "audio/wav").
	ContentType string
	// Data holds the raw file bytes.
	Data []byte
}

// Bytes builds an Attachment from raw bytes (name "audio.wav").
func Bytes(data []byte) Attachment {
	return Attachment{Name: "audio.wav", ContentType: "audio/wav", Data: data}
}

// File reads a file from disk into an Attachment, guessing the MIME type from
// the extension.
func File(path string) (Attachment, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Attachment{}, err
	}
	return Attachment{
		Name:        filepath.Base(path),
		ContentType: mimeTypeFor(filepath.Ext(path)),
		Data:        data,
	}, nil
}

// WithName overrides the file name of an attachment.
func (a Attachment) WithName(name string) Attachment {
	a.Name = name
	return a
}

// WithContentType overrides the MIME type of an attachment.
func (a Attachment) WithContentType(contentType string) Attachment {
	a.ContentType = contentType
	return a
}

func mimeTypeFor(ext string) string {
	switch strings.ToLower(ext) {
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".ogg", ".opus":
		return "audio/ogg"
	case ".m4a", ".aac":
		return "audio/aac"
	case ".flac":
		return "audio/flac"
	case ".mp4":
		return "video/mp4"
	case ".mov":
		return "video/quicktime"
	case ".webm":
		return "video/webm"
	default:
		return "application/octet-stream"
	}
}

// ───────────────────────────── transport ──────────────────────────────────

func (c *Client) send(ctx context.Context, method, path string, query url.Values, contentType string, body io.Reader) ([]byte, error) {
	req, err := c.newRequest(ctx, method, path, query, contentType, body)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, newAPIError(resp.StatusCode, data)
	}
	return data, nil
}

func (c *Client) requestJSON(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(encoded)
	}
	data, err := c.send(ctx, method, path, query, "application/json", reader)
	if err != nil {
		return err
	}
	return decodeInto(data, out)
}

func (c *Client) getJSON(ctx context.Context, path string, out any) error {
	return c.requestJSON(ctx, http.MethodGet, path, nil, nil, out)
}

func (c *Client) postJSON(ctx context.Context, path string, body any, out any) error {
	return c.requestJSON(ctx, http.MethodPost, path, nil, body, out)
}

// postMultipart uploads files (one entry per part, so the same field name may
// repeat) and form fields.
func (c *Client) postMultipart(ctx context.Context, path string, files []fileField, fields map[string]string, query url.Values, out any) error {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for _, file := range files {
		att := file.att
		name := att.Name
		if name == "" {
			name = file.key
		}
		contentType := att.ContentType
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		header := textproto.MIMEHeader{}
		header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, file.key, name))
		header.Set("Content-Type", contentType)
		part, err := writer.CreatePart(header)
		if err != nil {
			return err
		}
		if _, err := part.Write(att.Data); err != nil {
			return err
		}
	}
	for _, key := range sortedKeys(fields) {
		if err := writer.WriteField(key, fields[key]); err != nil {
			return err
		}
	}
	if err := writer.Close(); err != nil {
		return err
	}

	data, err := c.send(ctx, http.MethodPost, path, query, writer.FormDataContentType(), &buf)
	if err != nil {
		return err
	}
	return decodeInto(data, out)
}

// getBytes downloads a binary payload (or a text payload as raw bytes).
func (c *Client) getBytes(ctx context.Context, path string, query url.Values) ([]byte, error) {
	return c.send(ctx, http.MethodGet, path, query, "", nil)
}

func (c *Client) wsURL(path string, query url.Values) (string, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	default:
		u.Scheme = "ws"
	}
	u.Path = path
	u.RawQuery = query.Encode()
	return u.String(), nil
}

// ───────────────────────────── helpers ────────────────────────────────────

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// joinCSV renders a string slice as the comma-separated form the API expects
// (used by keyterms and profile_ids).
func joinCSV(values []string) string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			filtered = append(filtered, trimmed)
		}
	}
	return strings.Join(filtered, ",")
}

// EffectsJSON encodes an effect chain into the JSON string the API expects.
func EffectsJSON(effects []map[string]any) (string, error) {
	encoded, err := json.Marshal(effects)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

// B64 encodes audio bytes as base64 for inline batch items.
func B64(data []byte) string {
	return base64Encode(data)
}

// B64File reads a file and encodes it as base64 for inline batch items.
func B64File(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return base64Encode(data), nil
}

func itoa(value int) string { return strconv.Itoa(value) }
