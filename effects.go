package voicekit

import (
	"context"
	"encoding/json"
)

// ──────────────────────────── audio effects ────────────────────────────────

// AudioEffectsOptions controls POST /v1/audio/effects.
type AudioEffectsOptions struct {
	// Effects is the effect chain, e.g. {[]map[string]any{{"type": "reverb", "room_size": 0.5}}}
	Effects []map[string]any
	// OutputFormat is "wav" (default), "mp3" or "ogg".
	OutputFormat string
	// WebhookURL receives the result when the job completes.
	WebhookURL string
}

// ApplyAudioEffects starts an async audio-effects job; poll it with
// GetAudioEffectsJob and fetch the result with DownloadAudioEffects.
func (c *Client) ApplyAudioEffects(ctx context.Context, audio Attachment, opts *AudioEffectsOptions) (Object, error) {
	var effects []map[string]any
	if opts != nil {
		effects = opts.Effects
	}
	encoded, err := json.Marshal(effectsOrEmpty(effects))
	if err != nil {
		return nil, err
	}
	fields := map[string]string{"effects": string(encoded)}
	if opts != nil {
		if opts.OutputFormat != "" {
			fields["output_format"] = opts.OutputFormat
		}
		if opts.WebhookURL != "" {
			fields["webhookUrl"] = opts.WebhookURL
		}
	}
	var out Object
	if err := c.postMultipart(ctx, "/v1/audio/effects", []fileField{{"audio", audio}}, fields, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetAudioEffectsJob polls an audio-effects job.
func (c *Client) GetAudioEffectsJob(ctx context.Context, jobID string) (Object, error) {
	var out Object
	err := c.getJSON(ctx, "/v1/audio/effects/"+jobID, &out)
	return out, err
}

// DownloadAudioEffects downloads the processed audio of a completed job.
func (c *Client) DownloadAudioEffects(ctx context.Context, jobID string) ([]byte, error) {
	return c.getBytes(ctx, "/v1/audio/effects/"+jobID+"/audio", nil)
}

// ──────────────────────────── video effects ────────────────────────────────

// VideoEffectsOptions controls POST /v1/video/effects.
type VideoEffectsOptions struct {
	Effects []map[string]any
	// Mode is "mux" (default, video output) or "audio" (processed audio track).
	Mode string
	// Audio optionally replaces the video's audio track (mux mode only).
	Audio *Attachment
	// OutputFormat overrides the container of the produced artifact.
	OutputFormat string
	WebhookURL   string
}

// ApplyVideoEffects starts an async video-effects job; poll it with
// GetVideoEffectsJob and fetch the artifact with DownloadVideoEffects.
func (c *Client) ApplyVideoEffects(ctx context.Context, video Attachment, opts *VideoEffectsOptions) (Object, error) {
	var effects []map[string]any
	if opts != nil {
		effects = opts.Effects
	}
	encoded, err := json.Marshal(effectsOrEmpty(effects))
	if err != nil {
		return nil, err
	}
	files := []fileField{{"video", video.WithContentType("video/mp4")}}
	fields := map[string]string{"effects": string(encoded)}
	if opts != nil {
		if opts.Audio != nil {
			files = append(files, fileField{"audio", *opts.Audio})
		}
		if opts.Mode != "" {
			fields["mode"] = opts.Mode
		}
		if opts.OutputFormat != "" {
			fields["output_format"] = opts.OutputFormat
		}
		if opts.WebhookURL != "" {
			fields["webhookUrl"] = opts.WebhookURL
		}
	}
	var out Object
	if err := c.postMultipart(ctx, "/v1/video/effects", files, fields, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetVideoEffectsJob polls a video-effects job.
func (c *Client) GetVideoEffectsJob(ctx context.Context, jobID string) (Object, error) {
	var out Object
	err := c.getJSON(ctx, "/v1/video/effects/"+jobID, &out)
	return out, err
}

// DownloadVideoEffects downloads the produced artifact (video or audio).
func (c *Client) DownloadVideoEffects(ctx context.Context, jobID string) ([]byte, error) {
	return c.getBytes(ctx, "/v1/video/effects/"+jobID+"/file", nil)
}

// ──────────────────────────── audio cleaning ───────────────────────────────

// CleanAudioOptions controls POST /v1/audio/clean.
type CleanAudioOptions struct {
	// Options is the cleaning preset; nil applies denoise + normalize.
	//
	//	{"denoise": {"strength": 0.8, "stationary": true},
	//	 "normalize": {"target_db": -1.0}, "high_pass": 80, "low_pass": 12000}
	Options map[string]any
	// OutputFormat is "wav" (default), "mp3" or "ogg".
	OutputFormat string
	WebhookURL   string
}

// CleanAudio starts an async audio-cleaning job; poll it with
// GetAudioCleaningJob and fetch the result with DownloadAudioCleaning.
func (c *Client) CleanAudio(ctx context.Context, audio Attachment, opts *CleanAudioOptions) (Object, error) {
	fields := map[string]string{}
	if opts != nil {
		if opts.OutputFormat != "" {
			fields["output_format"] = opts.OutputFormat
		}
		if opts.WebhookURL != "" {
			fields["webhookUrl"] = opts.WebhookURL
		}
		if opts.Options != nil {
			encoded, err := json.Marshal(opts.Options)
			if err != nil {
				return nil, err
			}
			fields["options"] = string(encoded)
		}
	}
	var out Object
	if err := c.postMultipart(ctx, "/v1/audio/clean", []fileField{{"audio", audio}}, fields, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetAudioCleaningJob polls an audio-cleaning job.
func (c *Client) GetAudioCleaningJob(ctx context.Context, jobID string) (Object, error) {
	var out Object
	err := c.getJSON(ctx, "/v1/audio/clean/"+jobID, &out)
	return out, err
}

// DownloadAudioCleaning downloads the cleaned audio of a completed job.
func (c *Client) DownloadAudioCleaning(ctx context.Context, jobID string) ([]byte, error) {
	return c.getBytes(ctx, "/v1/audio/clean/"+jobID+"/audio", nil)
}

// ────────────────────────────── batches ───────────────────────────────────

// BatchSynthesize queues a batch of synthesis requests; each item is a
// /v1/synthesize body. Poll the batch with GetBatch.
func (c *Client) BatchSynthesize(ctx context.Context, items []map[string]any) (Object, error) {
	var out Object
	err := c.postJSON(ctx, "/v1/batch/synthesize", map[string]any{"items": items}, &out)
	return out, err
}

// BatchAnalyze queues a batch of analysis requests; each item needs an inline
// base64 "audio" field (see B64 and B64File).
func (c *Client) BatchAnalyze(ctx context.Context, items []map[string]any) (Object, error) {
	var out Object
	err := c.postJSON(ctx, "/v1/batch/analyze", map[string]any{"items": items}, &out)
	return out, err
}

// GetBatch polls a batch job.
func (c *Client) GetBatch(ctx context.Context, batchID string) (Object, error) {
	var out Object
	err := c.getJSON(ctx, "/v1/batch/"+batchID, &out)
	return out, err
}

// ────────────────────────────── account ───────────────────────────────────

// Usage returns the current monthly usage for the authenticated key.
func (c *Client) Usage(ctx context.Context) (Object, error) {
	var out Object
	err := c.getJSON(ctx, "/v1/usage", &out)
	return out, err
}

// BillingBalance returns the current balance, plan and recent transactions.
func (c *Client) BillingBalance(ctx context.Context) (Object, error) {
	var out Object
	err := c.getJSON(ctx, "/v1/billing/balance", &out)
	return out, err
}

func effectsOrEmpty(effects []map[string]any) []map[string]any {
	if effects == nil {
		return []map[string]any{}
	}
	return effects
}
