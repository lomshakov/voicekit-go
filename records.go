package voicekit

import (
	"context"
	"net/http"
	"net/url"
)

// ───────────────────────────── recordings ─────────────────────────────────

// RecordingListOptions filters GET /v1/recordings.
type RecordingListOptions struct {
	// Source is one of upload, link, bot, stream.
	Source string
	Tag    string
	Folder string
	Limit  int
	Offset int
}

// ListRecordings lists the caller's recordings (newest first).
func (c *Client) ListRecordings(ctx context.Context, opts *RecordingListOptions) (Object, error) {
	params := url.Values{}
	if opts != nil {
		if opts.Source != "" {
			params.Set("source", opts.Source)
		}
		if opts.Tag != "" {
			params.Set("tag", opts.Tag)
		}
		if opts.Folder != "" {
			params.Set("folder", opts.Folder)
		}
		if opts.Limit > 0 {
			params.Set("limit", itoa(opts.Limit))
		}
		if opts.Offset > 0 {
			params.Set("offset", itoa(opts.Offset))
		}
	}
	var out Object
	err := c.requestJSON(ctx, http.MethodGet, "/v1/recordings", params, nil, &out)
	return out, err
}

// RecordingTags returns the distinct tags across the caller's recordings.
func (c *Client) RecordingTags(ctx context.Context) ([]string, error) {
	var out Object
	if err := c.getJSON(ctx, "/v1/recordings/tags", &out); err != nil {
		return nil, err
	}
	return out.Strings("values"), nil
}

// RecordingFolders returns the distinct folders across the caller's recordings.
func (c *Client) RecordingFolders(ctx context.Context) ([]string, error) {
	var out Object
	if err := c.getJSON(ctx, "/v1/recordings/folders", &out); err != nil {
		return nil, err
	}
	return out.Strings("values"), nil
}

// RecordingFromLink starts a diarized transcription from a public audio URL;
// the recording is tagged with source "link".
func (c *Client) RecordingFromLink(ctx context.Context, audioURL, language string) (Object, error) {
	body := map[string]any{"url": audioURL}
	setStr(body, "language", language)
	var out Object
	err := c.postJSON(ctx, "/v1/recordings/from-link", body, &out)
	return out, err
}

// GetRecording returns a recording's metadata.
func (c *Client) GetRecording(ctx context.Context, recordingID string) (Object, error) {
	var out Object
	err := c.getJSON(ctx, "/v1/recordings/"+recordingID, &out)
	return out, err
}

// RecordingTranscript returns a recording's transcript.
func (c *Client) RecordingTranscript(ctx context.Context, recordingID string) (Object, error) {
	var out Object
	err := c.getJSON(ctx, "/v1/recordings/"+recordingID+"/transcript", &out)
	return out, err
}

// RecordingSpeakers returns a recording's speakers, timeline and metrics.
func (c *Client) RecordingSpeakers(ctx context.Context, recordingID string) (Object, error) {
	var out Object
	err := c.getJSON(ctx, "/v1/recordings/"+recordingID+"/speakers", &out)
	return out, err
}

// UpdateRecordingOptions patches a recording's tags and folder. Zero-valued
// fields are left untouched server-side.
type UpdateRecordingOptions struct {
	// Tags replaces the tag set; nil leaves it untouched.
	Tags []string
	// ClearTags removes every tag (wins over Tags).
	ClearTags bool
	// Folder moves the recording; nil leaves it untouched, a pointer to ""
	// clears the folder.
	Folder *string
}

// UpdateRecording updates a recording's tags and/or folder.
func (c *Client) UpdateRecording(ctx context.Context, recordingID string, opts UpdateRecordingOptions) (Object, error) {
	body := map[string]any{}
	switch {
	case opts.ClearTags:
		body["tags"] = nil
	case opts.Tags != nil:
		body["tags"] = opts.Tags
	}
	if opts.Folder != nil {
		body["folder"] = *opts.Folder
	}
	var out Object
	err := c.requestJSON(ctx, http.MethodPatch, "/v1/recordings/"+recordingID, nil, body, &out)
	return out, err
}

// UpdateSpeaker renames a diarized speaker or assigns a role
// (operator, client, participant).
func (c *Client) UpdateSpeaker(ctx context.Context, recordingID, speakerID, displayName, role string) (Object, error) {
	body := map[string]any{}
	setStr(body, "display_name", displayName)
	setStr(body, "role", role)
	var out Object
	err := c.requestJSON(ctx, http.MethodPatch,
		"/v1/recordings/"+recordingID+"/speakers/"+speakerID, nil, body, &out)
	return out, err
}

// DeleteRecording deletes a recording and its stored audio.
func (c *Client) DeleteRecording(ctx context.Context, recordingID string) error {
	_, err := c.send(ctx, http.MethodDelete, "/v1/recordings/"+recordingID, nil, "", nil)
	return err
}

// DownloadRecordingAudio downloads a recording's stored audio.
func (c *Client) DownloadRecordingAudio(ctx context.Context, recordingID string) ([]byte, error) {
	return c.getBytes(ctx, "/v1/recordings/"+recordingID+"/audio", nil)
}

// ExportRecording exports a transcript as txt, md, srt, vtt, docx or pdf.
func (c *Client) ExportRecording(ctx context.Context, recordingID, format string) ([]byte, error) {
	if format == "" {
		format = "txt"
	}
	return c.getBytes(ctx, "/v1/recordings/"+recordingID+"/export", queryParams(map[string]string{"format": format}))
}

// ──────────────────────────── share links ─────────────────────────────────

// CreateShareOptions controls POST /v1/recordings/{id}/share.
type CreateShareOptions struct {
	// ExpiresInSeconds limits the link lifetime (0 = no expiry).
	ExpiresInSeconds int
	// Password protects the link.
	Password string
}

// CreateShare creates a public share link for a recording.
func (c *Client) CreateShare(ctx context.Context, recordingID string, opts *CreateShareOptions) (Object, error) {
	body := map[string]any{}
	if opts != nil {
		setInt(body, "expires_in_seconds", opts.ExpiresInSeconds)
		setStr(body, "password", opts.Password)
	}
	var out Object
	err := c.postJSON(ctx, "/v1/recordings/"+recordingID+"/share", body, &out)
	return out, err
}

// ListShares lists the active share links of a recording.
func (c *Client) ListShares(ctx context.Context, recordingID string) ([]Object, error) {
	data, err := c.send(ctx, http.MethodGet, "/v1/recordings/"+recordingID+"/share", nil, "", nil)
	if err != nil {
		return nil, err
	}
	return decodeList(data)
}

// RevokeShare revokes a share link.
func (c *Client) RevokeShare(ctx context.Context, recordingID, token string) error {
	_, err := c.send(ctx, http.MethodDelete, "/v1/recordings/"+recordingID+"/share/"+token, nil, "", nil)
	return err
}

// ─────────────────────────────── call QA ──────────────────────────────────

// QAEvaluateOptions controls POST /v1/qa/evaluate.
type QAEvaluateOptions struct {
	// Checklist holds items such as
	// {"id": "greeting", "kind": "required", "description": "...", "weight": 1.0}.
	Checklist []map[string]any
	// WebhookURL receives a qa.violation event when the call is flagged.
	WebhookURL string
}

// QAEvaluate scores a recording against a QA checklist (Pro/Business).
func (c *Client) QAEvaluate(ctx context.Context, recordingID string, opts *QAEvaluateOptions) (Object, error) {
	body := map[string]any{"recording_id": recordingID}
	if opts != nil {
		body["checklist"] = opts.Checklist
		setStr(body, "webhook_url", opts.WebhookURL)
	}
	var out Object
	err := c.postJSON(ctx, "/v1/qa/evaluate", body, &out)
	return out, err
}

// QAAnalytics returns the score trend, top violations and score by operator.
func (c *Client) QAAnalytics(ctx context.Context, days int) (Object, error) {
	if days <= 0 {
		days = 30
	}
	var out Object
	err := c.requestJSON(ctx, http.MethodGet, "/v1/qa/analytics",
		queryParams(map[string]string{"days": itoa(days)}), nil, &out)
	return out, err
}

// QAEvaluations lists persisted QA evaluations (newest first).
func (c *Client) QAEvaluations(ctx context.Context, limit, offset int) (Object, error) {
	if limit <= 0 {
		limit = 20
	}
	var out Object
	err := c.requestJSON(ctx, http.MethodGet, "/v1/qa/evaluations", queryParams(map[string]string{
		"limit":  itoa(limit),
		"offset": itoa(offset),
	}), nil, &out)
	return out, err
}

// QAExport exports QA evaluations as CSV or JSON for CRM import.
func (c *Client) QAExport(ctx context.Context, format string, days int) ([]byte, error) {
	if format == "" {
		format = "csv"
	}
	if days <= 0 {
		days = 30
	}
	return c.getBytes(ctx, "/v1/qa/evaluations/export", queryParams(map[string]string{
		"format": format,
		"days":   itoa(days),
	}))
}

// ──────────────────────── search & meeting notes ──────────────────────────

// SearchOptions filters POST /v1/search.
type SearchOptions struct {
	Limit int
	// Keywords enables full-text ranking on top of semantic similarity.
	Keywords string
	// Source is one of upload, link, bot, stream.
	Source string
	// Speaker is a diarization label such as SPEAKER_00.
	Speaker string
	// From and To are ISO-8601 timestamps (UTC).
	From string
	To   string
	// MinDuration and MaxDuration bound the recording length in seconds.
	MinDuration float64
	MaxDuration float64
}

// Search runs a hybrid semantic/full-text search over recordings
// (Pro/Business).
func (c *Client) Search(ctx context.Context, query string, opts *SearchOptions) (Object, error) {
	body := map[string]any{"query": query}
	if opts != nil {
		setInt(body, "limit", opts.Limit)
		setStr(body, "keywords", opts.Keywords)
		setStr(body, "source", opts.Source)
		setStr(body, "speaker", opts.Speaker)
		setStr(body, "from", opts.From)
		setStr(body, "to", opts.To)
		setFloat(body, "min_duration_seconds", opts.MinDuration)
		setFloat(body, "max_duration_seconds", opts.MaxDuration)
	}
	var out Object
	err := c.postJSON(ctx, "/v1/search", body, &out)
	return out, err
}

// Ask answers a question over the recording library (RAG) with verbatim
// citations.
func (c *Client) Ask(ctx context.Context, query string) (Object, error) {
	var out Object
	err := c.postJSON(ctx, "/v1/ask", map[string]any{"query": query}, &out)
	return out, err
}

// MeetingProtocol generates a meeting protocol from a recording transcript.
// Template is one of custom, standup, demo, interview, retro, one_on_one.
func (c *Client) MeetingProtocol(ctx context.Context, recordingID, template string) (Object, error) {
	if template == "" {
		template = "custom"
	}
	body := map[string]any{"recording_id": recordingID, "template": template}
	var out Object
	err := c.postJSON(ctx, "/v1/meetings/protocol", body, &out)
	return out, err
}
