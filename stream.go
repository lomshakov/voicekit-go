package voicekit

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/coder/websocket"
)

// ErrStreamClosed is returned by Stream.Receive once the server closes the
// session.
var ErrStreamClosed = errors.New("voicekit: stream closed")

// StreamOptions controls a streaming transcription session.
type StreamOptions struct {
	// Language is an ISO-639-1 hint; empty = auto-detect.
	Language string
	// Keyterms biases recognition towards domain vocabulary.
	Keyterms []string
	// Interim enables interim results (server default is true).
	Interim *bool
}

// Stream is a bidirectional VoiceKit WebSocket session. It carries raw PCM16
// (16 kHz, mono, little-endian) binary frames up and JSON events
// (session, vad, partial, final, error) down. It is not safe for concurrent
// use; call Send and Receive from the same goroutine or guard them yourself.
type Stream struct {
	conn *websocket.Conn
}

// TranscribeStream opens WS /v1/transcribe/stream (Pro/Business).
func (c *Client) TranscribeStream(ctx context.Context, opts *StreamOptions) (*Stream, error) {
	params := url.Values{}
	if opts != nil {
		if opts.Language != "" {
			params.Set("language", opts.Language)
		}
		if keyterms := joinCSV(opts.Keyterms); keyterms != "" {
			params.Set("keyterms", keyterms)
		}
		if opts.Interim != nil {
			params.Set("interim", boolString(*opts.Interim))
		}
	}
	return c.dialStream(ctx, "/v1/transcribe/stream", params)
}

// VADStream opens WS /v1/vad/stream for speech turn detection without
// recognition.
func (c *Client) VADStream(ctx context.Context) (*Stream, error) {
	return c.dialStream(ctx, "/v1/vad/stream", nil)
}

func (c *Client) dialStream(ctx context.Context, path string, params url.Values) (*Stream, error) {
	endpoint, err := c.wsURL(path, params)
	if err != nil {
		return nil, err
	}
	conn, resp, err := websocket.Dial(ctx, endpoint, &websocket.DialOptions{
		HTTPHeader: http.Header{
			"X-Api-Key":  []string{c.apiKey},
			"User-Agent": []string{"voicekit-go/" + Version},
		},
	})
	if err != nil {
		if resp != nil && resp.Body != nil {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != 0 && resp.StatusCode != http.StatusSwitchingProtocols {
				return nil, newAPIError(resp.StatusCode, body)
			}
		}
		return nil, err
	}
	return &Stream{conn: conn}, nil
}

// SendAudio writes a raw PCM16 frame (16 kHz, mono, little-endian).
func (s *Stream) SendAudio(ctx context.Context, pcm16 []byte) error {
	return s.conn.Write(ctx, websocket.MessageBinary, pcm16)
}

// SendJSON writes a control message; maps and structs are JSON-encoded.
func (s *Stream) SendJSON(ctx context.Context, message any) error {
	var data []byte
	switch value := message.(type) {
	case string:
		data = []byte(value)
	case []byte:
		data = value
	default:
		encoded, err := json.Marshal(message)
		if err != nil {
			return err
		}
		data = encoded
	}
	return s.conn.Write(ctx, websocket.MessageText, data)
}

// Stop signals the end of speech so the server finalizes the utterance.
func (s *Stream) Stop(ctx context.Context) error {
	return s.SendJSON(ctx, map[string]any{"type": "stop"})
}

// Receive reads the next JSON event. It returns ErrStreamClosed after a normal
// server-side close.
func (s *Stream) Receive(ctx context.Context) (Object, error) {
	_, data, err := s.conn.Read(ctx)
	if err != nil {
		if websocket.CloseStatus(err) != -1 || errors.Is(err, io.EOF) {
			return nil, ErrStreamClosed
		}
		return nil, err
	}
	var out Object
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Close closes the WebSocket with a normal-closure status.
func (s *Stream) Close() error {
	return s.conn.Close(websocket.StatusNormalClosure, "")
}
