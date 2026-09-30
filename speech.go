package voicekit

import (
	"bytes"
	"context"
)

// ─────────────────────────── transcription ────────────────────────────────

// TranscribeOptions controls POST /v1/transcribe (async job).
type TranscribeOptions struct {
	// Language is an ISO-639-1 hint; empty = auto-detect.
	Language string
	// Diarization labels speakers (Basic and above).
	Diarization bool
	// Keyterms biases recognition towards domain vocabulary.
	Keyterms []string
	// Clean runs denoise + normalize before recognition.
	Clean bool
	// LongForm enables long-form transcription (up to 4 h / 512 MB).
	LongForm bool
	// WebhookURL receives the result when the job completes.
	WebhookURL string
}

func (o *TranscribeOptions) fields() map[string]string {
	fields := map[string]string{}
	if o == nil {
		return fields
	}
	if o.Language != "" {
		fields["language"] = o.Language
	}
	if o.Diarization {
		fields["diarization"] = "true"
	}
	if o.Clean {
		fields["clean"] = "true"
	}
	if o.LongForm {
		fields["longForm"] = "true"
	}
	if o.WebhookURL != "" {
		fields["webhookUrl"] = o.WebhookURL
	}
	if keyterms := joinCSV(o.Keyterms); keyterms != "" {
		fields["keyterms"] = keyterms
	}
	return fields
}

// Transcribe starts an async transcription job; poll it with
// GetTranscriptionJob.
func (c *Client) Transcribe(ctx context.Context, audio Attachment, opts *TranscribeOptions) (Object, error) {
	var out Object
	err := c.postMultipart(ctx, "/v1/transcribe", []fileField{{"audio", audio}}, opts.fields(), nil, &out)
	return out, err
}

// TranscribeSyncOptions controls POST /v1/transcribe/sync (files up to 3 min).
type TranscribeSyncOptions struct {
	Language    string
	Diarization bool
	Keyterms    []string
	Clean       bool
}

// TranscribeSync transcribes a short file synchronously.
func (c *Client) TranscribeSync(ctx context.Context, audio Attachment, opts *TranscribeSyncOptions) (Object, error) {
	fields := map[string]string{}
	if opts != nil {
		if opts.Language != "" {
			fields["language"] = opts.Language
		}
		if opts.Diarization {
			fields["diarization"] = "true"
		}
		if opts.Clean {
			fields["clean"] = "true"
		}
		if keyterms := joinCSV(opts.Keyterms); keyterms != "" {
			fields["keyterms"] = keyterms
		}
	}
	var out Object
	err := c.postMultipart(ctx, "/v1/transcribe/sync", []fileField{{"audio", audio}}, fields, nil, &out)
	return out, err
}

// GetTranscriptionJob polls a transcription job.
func (c *Client) GetTranscriptionJob(ctx context.Context, jobID string) (Object, error) {
	var out Object
	err := c.getJSON(ctx, "/v1/transcribe/"+jobID, &out)
	return out, err
}

// SubtitleOptions controls GET /v1/transcribe/{id}/subtitles.
type SubtitleOptions struct {
	// Format is "vtt" (default) or "srt".
	Format string
	// TargetLanguage translates the captions (e.g. "en").
	TargetLanguage string
	// HotMarks marks fast or unclear speech.
	HotMarks bool
}

// Subtitles downloads VTT/SRT captions for a completed transcription job.
func (c *Client) Subtitles(ctx context.Context, jobID string, opts *SubtitleOptions) (string, error) {
	params := map[string]string{}
	if opts != nil {
		if opts.Format != "" {
			params["format"] = opts.Format
		}
		params["target_language"] = opts.TargetLanguage
		if opts.HotMarks {
			params["hot_marks"] = "true"
		}
	}
	data, err := c.getBytes(ctx, "/v1/transcribe/"+jobID+"/subtitles", queryParams(params))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// TranslateTranscript translates a completed transcript, preserving timestamps
// and speakers. Billed against the plan's LLM token budget.
func (c *Client) TranslateTranscript(ctx context.Context, jobID, targetLanguage string) (Object, error) {
	var out Object
	err := c.postJSON(ctx, "/v1/transcribe/"+jobID+"/translate", map[string]any{"target_language": targetLanguage}, &out)
	return out, err
}

// VAD detects speech segments in an audio file (Silero VAD).
func (c *Client) VAD(ctx context.Context, audio Attachment) (Object, error) {
	var out Object
	err := c.postMultipart(ctx, "/v1/vad", []fileField{{"audio", audio}}, nil, nil, &out)
	return out, err
}

// ────────────────────────────── analysis ──────────────────────────────────

// AnalyzeOptions controls POST /v1/analyze (async job).
type AnalyzeOptions struct {
	Language    string
	Diarization bool
	Keyterms    []string
	Clean       bool
	WebhookURL  string
}

// Analyze starts an async analysis job (emotions, keywords, entities,
// optionally per speaker); poll it with GetAnalysisJob.
func (c *Client) Analyze(ctx context.Context, audio Attachment, opts *AnalyzeOptions) (Object, error) {
	fields := map[string]string{}
	if opts != nil {
		if opts.Language != "" {
			fields["language"] = opts.Language
		}
		if opts.Diarization {
			fields["diarization"] = "true"
		}
		if opts.Clean {
			fields["clean"] = "true"
		}
		if opts.WebhookURL != "" {
			fields["webhookUrl"] = opts.WebhookURL
		}
		if keyterms := joinCSV(opts.Keyterms); keyterms != "" {
			fields["keyterms"] = keyterms
		}
	}
	var out Object
	err := c.postMultipart(ctx, "/v1/analyze", []fileField{{"audio", audio}}, fields, nil, &out)
	return out, err
}

// AnalyzeSyncOptions controls POST /v1/analyze/sync (files up to 3 min).
type AnalyzeSyncOptions struct {
	Language    string
	Diarization bool
	// Emotions, Keywords and Entities default to true server-side; set the
	// pointers to disable an individual extractor.
	Emotions *bool
	Keywords *bool
	Entities *bool
	Keyterms []string
	Clean    bool
}

// AnalyzeSync analyses a short file synchronously.
func (c *Client) AnalyzeSync(ctx context.Context, audio Attachment, opts *AnalyzeSyncOptions) (Object, error) {
	fields := map[string]string{}
	if opts != nil {
		if opts.Language != "" {
			fields["language"] = opts.Language
		}
		if opts.Diarization {
			fields["diarization"] = "true"
		}
		if opts.Clean {
			fields["clean"] = "true"
		}
		if opts.Emotions != nil {
			fields["emotions"] = boolString(*opts.Emotions)
		}
		if opts.Keywords != nil {
			fields["keywords"] = boolString(*opts.Keywords)
		}
		if opts.Entities != nil {
			fields["entities"] = boolString(*opts.Entities)
		}
		if keyterms := joinCSV(opts.Keyterms); keyterms != "" {
			fields["keyterms"] = keyterms
		}
	}
	var out Object
	err := c.postMultipart(ctx, "/v1/analyze/sync", []fileField{{"audio", audio}}, fields, nil, &out)
	return out, err
}

// GetAnalysisJob polls an analysis job.
func (c *Client) GetAnalysisJob(ctx context.Context, jobID string) (Object, error) {
	var out Object
	err := c.getJSON(ctx, "/v1/analyze/"+jobID, &out)
	return out, err
}

// ────────────────────────── text intelligence ──────────────────────────────

// DetectLanguage identifies the language of a raw text.
func (c *Client) DetectLanguage(ctx context.Context, text string) (Object, error) {
	var out Object
	err := c.postJSON(ctx, "/v1/detect-language", map[string]any{"text": text}, &out)
	return out, err
}

// Redact masks PII (names, phones, addresses, card numbers) in raw text.
func (c *Client) Redact(ctx context.Context, text, language string) (Object, error) {
	body := map[string]any{"text": text}
	setStr(body, "language", language)
	var out Object
	err := c.postJSON(ctx, "/v1/redact", body, &out)
	return out, err
}

// Topics extracts key topics from raw text.
func (c *Client) Topics(ctx context.Context, text, language string) (Object, error) {
	body := map[string]any{"text": text}
	setStr(body, "language", language)
	var out Object
	err := c.postJSON(ctx, "/v1/analyze/topics", body, &out)
	return out, err
}

// SummarizeOptions controls POST /v1/analyze/summarize.
type SummarizeOptions struct {
	Language string
	// MaxSentences caps the summary length (0 = server default).
	MaxSentences int
}

// Summarize condenses raw text into a short summary.
func (c *Client) Summarize(ctx context.Context, text string, opts *SummarizeOptions) (Object, error) {
	body := map[string]any{"text": text}
	if opts != nil {
		setStr(body, "language", opts.Language)
		setInt(body, "max_sentences", opts.MaxSentences)
	}
	var out Object
	err := c.postJSON(ctx, "/v1/analyze/summarize", body, &out)
	return out, err
}

// Moderate flags profanity, insults and hate speech in raw text.
func (c *Client) Moderate(ctx context.Context, text, language string) (Object, error) {
	body := map[string]any{"text": text}
	setStr(body, "language", language)
	var out Object
	err := c.postJSON(ctx, "/v1/moderate", body, &out)
	return out, err
}

// ──────────────────────────── evaluation ──────────────────────────────────

// EvaluateOptions controls POST /v1/eval.
type EvaluateOptions struct {
	Language string
	// Normalize ignores case and punctuation; nil = server default.
	Normalize *bool
}

// Evaluate measures speech quality (word error rate) against a reference text.
func (c *Client) Evaluate(ctx context.Context, audio Attachment, reference string, opts *EvaluateOptions) (Object, error) {
	fields := map[string]string{"reference": reference}
	if opts != nil {
		if opts.Language != "" {
			fields["language"] = opts.Language
		}
		if opts.Normalize != nil {
			fields["normalize"] = boolString(*opts.Normalize)
		}
	}
	var out Object
	err := c.postMultipart(ctx, "/v1/eval", []fileField{{"audio", audio}}, fields, nil, &out)
	return out, err
}

// ────────────────────────────── internals ─────────────────────────────────

type fileField struct {
	key string
	att Attachment
}

func bytesReader(data []byte) *bytes.Reader { return bytes.NewReader(data) }

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
