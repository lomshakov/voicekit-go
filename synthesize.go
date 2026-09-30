package voicekit

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
)

// SynthesizeOptions controls POST /v1/synthesize and /v1/synthesize/stream.
type SynthesizeOptions struct {
	// Voice is a preset id (e.g. "preset_anna") or a cloned voice id.
	Voice string
	// Format is mp3 (default), wav or ogg.
	Format string
	// SampleRate is the output sample rate in Hz.
	SampleRate int
	// Speed is a multiplier, 1.0 = normal.
	Speed float64
	// Pitch shifts the voice in semitones.
	Pitch float64
	// Emotion selects a preset emotion on supported voices.
	Emotion string
	// SSML switches the input to SSML markup.
	SSML *bool
	// PutAccent and PutYo are accepted for compatibility and do not affect
	// synthesis (the engine reads stress and "ё" on its own).
	PutAccent *bool
	PutYo     *bool
	// Normalize applies loudness normalisation to the result.
	Normalize *bool
	// Model is "standard" (default) or "premium" (clone, Pro/Business).
	Model string
	// Language overrides the synthesis language for clones.
	Language string
	// Effects is a JSON array of effect descriptors, see EffectsJSON.
	Effects string
}

func (o *SynthesizeOptions) body(text string) map[string]any {
	body := map[string]any{"text": text}
	if o == nil {
		return body
	}
	setStr(body, "voice", o.Voice)
	setStr(body, "format", o.Format)
	setInt(body, "sample_rate", o.SampleRate)
	setFloat(body, "speed", o.Speed)
	setFloat(body, "pitch", o.Pitch)
	setStr(body, "emotion", o.Emotion)
	setBool(body, "ssml", o.SSML)
	setBool(body, "put_accent", o.PutAccent)
	setBool(body, "put_yo", o.PutYo)
	setBool(body, "normalize", o.Normalize)
	setStr(body, "model", o.Model)
	setStr(body, "language", o.Language)
	setStr(body, "effects", o.Effects)
	return body
}

// Synthesize converts text to speech and returns the raw audio bytes.
func (c *Client) Synthesize(ctx context.Context, text string, opts *SynthesizeOptions) ([]byte, error) {
	encoded, err := json.Marshal(opts.body(text))
	if err != nil {
		return nil, err
	}
	data, err := c.send(ctx, http.MethodPost, "/v1/synthesize", nil, "application/json", bytesReader(encoded))
	if err != nil {
		return nil, err
	}
	return data, nil
}

// SynthesizeStream starts a streaming synthesis (Pro/Business) and returns the
// response body. Audio chunks arrive as the engine produces them; the caller
// must close the reader.
//
//	rc, err := client.SynthesizeStream(ctx, longText, nil)
//	if err != nil { return err }
//	defer rc.Close()
//	_, err = io.Copy(out, rc)
func (c *Client) SynthesizeStream(ctx context.Context, text string, opts *SynthesizeOptions) (io.ReadCloser, error) {
	encoded, err := json.Marshal(opts.body(text))
	if err != nil {
		return nil, err
	}
	req, err := c.newRequest(ctx, http.MethodPost, "/v1/synthesize/stream", nil, "application/json", bytesReader(encoded))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "*/*")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, newAPIError(resp.StatusCode, body)
	}
	return resp.Body, nil
}

// SynthesizeAsyncOptions controls POST /v1/synthesize/async (long-form
// audiobook jobs). Long-form synthesis always returns WAV.
type SynthesizeAsyncOptions struct {
	Voice      string
	Format     string
	SampleRate int
	Speed      float64
	Model      string
	Language   string
	WebhookURL string
}

// SynthesizeAsync queues a long-form synthesis job. Poll it with
// GetSynthesisJob and fetch the audio with DownloadSynthesisAudio.
func (c *Client) SynthesizeAsync(ctx context.Context, text string, opts *SynthesizeAsyncOptions) (Object, error) {
	body := map[string]any{"text": text}
	if opts != nil {
		setStr(body, "voice", opts.Voice)
		setStr(body, "format", opts.Format)
		setInt(body, "sample_rate", opts.SampleRate)
		setFloat(body, "speed", opts.Speed)
		setStr(body, "model", opts.Model)
		setStr(body, "language", opts.Language)
	}
	var query map[string]string
	if opts != nil && opts.WebhookURL != "" {
		query = map[string]string{"webhookUrl": opts.WebhookURL}
	}
	var out Object
	err := c.requestJSON(ctx, http.MethodPost, "/v1/synthesize/async", queryParams(query), body, &out)
	return out, err
}

// GetSynthesisJob polls a long-form synthesis job and returns its manifest.
func (c *Client) GetSynthesisJob(ctx context.Context, jobID string) (Object, error) {
	var out Object
	err := c.getJSON(ctx, "/v1/synthesize/async/"+jobID, &out)
	return out, err
}

// DownloadSynthesisAudio downloads the produced WAV of a completed job.
func (c *Client) DownloadSynthesisAudio(ctx context.Context, jobID string) ([]byte, error) {
	return c.getBytes(ctx, "/v1/synthesize/async/"+jobID+"/audio", nil)
}

// Voices returns the voice catalog.
func (c *Client) Voices(ctx context.Context) ([]Object, error) {
	data, err := c.send(ctx, http.MethodGet, "/v1/voices", nil, "", nil)
	if err != nil {
		return nil, err
	}
	return decodeList(data)
}

// Voice returns one voice by id (e.g. "preset_anna").
func (c *Client) Voice(ctx context.Context, voiceID string) (Object, error) {
	var out Object
	err := c.getJSON(ctx, "/v1/voices/"+voiceID, &out)
	return out, err
}
