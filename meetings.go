package voicekit

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// MeetingTarget attaches a captured meeting to a CRM deal.
type MeetingTarget struct {
	DealID string
	Phone  string
	Email  string
}

// CustomProtocolOptions selects a saved template or an inline section schema.
type CustomProtocolOptions struct {
	TemplateID string
	Schema     map[string]any
	WebhookURL string
}

// IntegrationConnection is a provider credential to store.
type IntegrationConnection struct {
	Provider     string
	Secret       string
	DisplayName  string
	BaseURL      string
	RefreshToken string
	Scopes       string
}

// RecordingPush describes a CRM push of a recording artifact.
type RecordingPush struct {
	Provider  string
	Kind      string
	DealID    string
	Phone     string
	Email     string
	SourceURL string
}

func limitQuery(limit int) url.Values {
	if limit <= 0 {
		limit = 50
	}
	return url.Values{"limit": {strconv.Itoa(limit)}}
}

func compactMap(values map[string]any) map[string]any {
	out := make(map[string]any, len(values))
	for key, value := range values {
		switch typed := value.(type) {
		case string:
			if typed != "" {
				out[key] = typed
			}
		case nil:
			// skipped
		default:
			out[key] = value
		}
	}
	return out
}

// MeetingEvents lists automatically captured meetings, newest first.
func (c *Client) MeetingEvents(ctx context.Context, limit int) (Object, error) {
	var out Object
	err := c.requestJSON(ctx, http.MethodGet, "/v1/meetings/events", limitQuery(limit), nil, &out)
	return out, err
}

// MeetingEvent returns one captured meeting.
func (c *Client) MeetingEvent(ctx context.Context, eventID string) (Object, error) {
	var out Object
	err := c.getJSON(ctx, "/v1/meetings/events/"+url.PathEscape(eventID), &out)
	return out, err
}

// SetMeetingTarget points a captured meeting at a CRM deal.
func (c *Client) SetMeetingTarget(ctx context.Context, eventID string, target MeetingTarget) (Object, error) {
	var out Object
	body := compactMap(map[string]any{
		"deal_id": target.DealID,
		"phone":   target.Phone,
		"email":   target.Email,
	})
	err := c.requestJSON(ctx, http.MethodPut, "/v1/meetings/events/"+url.PathEscape(eventID)+"/target", nil, body, &out)
	return out, err
}

// RetryMeeting retries a failed capture now.
func (c *Client) RetryMeeting(ctx context.Context, eventID string) (Object, error) {
	var out Object
	err := c.postJSON(ctx, "/v1/meetings/events/"+url.PathEscape(eventID)+"/retry", map[string]any{}, &out)
	return out, err
}

// MeetingCalendar lists provider calendar events.
func (c *Client) MeetingCalendar(ctx context.Context, limit int) (Object, error) {
	var out Object
	err := c.requestJSON(ctx, http.MethodGet, "/v1/meetings/calendar", limitQuery(limit), nil, &out)
	return out, err
}

// SyncMeetingCalendar pulls the provider calendar immediately.
func (c *Client) SyncMeetingCalendar(ctx context.Context) (Object, error) {
	var out Object
	err := c.postJSON(ctx, "/v1/meetings/calendar/sync", map[string]any{}, &out)
	return out, err
}

// WritebackMeetingCalendar writes the protocol link back to the provider event.
func (c *Client) WritebackMeetingCalendar(ctx context.Context, eventID string) (Object, error) {
	var out Object
	err := c.postJSON(ctx, "/v1/meetings/events/"+url.PathEscape(eventID)+"/calendar", map[string]any{}, &out)
	return out, err
}

// Protocols lists persisted meeting protocols, newest first.
func (c *Client) Protocols(ctx context.Context, limit int) (Object, error) {
	var out Object
	err := c.requestJSON(ctx, http.MethodGet, "/v1/meetings/protocols", limitQuery(limit), nil, &out)
	return out, err
}

// Protocol returns one persisted protocol.
func (c *Client) Protocol(ctx context.Context, protocolID string) (Object, error) {
	var out Object
	err := c.getJSON(ctx, "/v1/meetings/protocols/"+url.PathEscape(protocolID), &out)
	return out, err
}

// ApproveProtocol approves a draft so it may be pushed to CRM.
func (c *Client) ApproveProtocol(ctx context.Context, protocolID string) (Object, error) {
	var out Object
	err := c.postJSON(ctx, "/v1/meetings/protocols/"+url.PathEscape(protocolID)+"/approve", map[string]any{}, &out)
	return out, err
}

// UpdateProtocol replaces the protocol Markdown.
func (c *Client) UpdateProtocol(ctx context.Context, protocolID, markdown string) (Object, error) {
	var out Object
	err := c.requestJSON(ctx, http.MethodPut, "/v1/meetings/protocols/"+url.PathEscape(protocolID), nil,
		map[string]any{"markdown": markdown}, &out)
	return out, err
}

// ExportProtocol downloads a protocol as md, pdf or docx.
func (c *Client) ExportProtocol(ctx context.Context, protocolID, format string) ([]byte, error) {
	if format == "" {
		format = "md"
	}
	query := url.Values{"format": {format}}
	return c.send(ctx, http.MethodGet, "/v1/meetings/protocols/"+url.PathEscape(protocolID)+"/export", query, "", nil)
}

// CustomProtocol builds a protocol against an inline schema or a saved template.
func (c *Client) CustomProtocol(ctx context.Context, recordingID string, opts CustomProtocolOptions) (Object, error) {
	var out Object
	body := compactMap(map[string]any{
		"template_id": opts.TemplateID,
		"webhook_url": opts.WebhookURL,
	})
	body["recording_id"] = recordingID
	if opts.Schema != nil {
		body["schema"] = opts.Schema
	}
	err := c.postJSON(ctx, "/v1/meetings/protocol/custom", body, &out)
	return out, err
}

// ProtocolTemplates lists saved protocol templates.
func (c *Client) ProtocolTemplates(ctx context.Context) (Object, error) {
	var out Object
	err := c.getJSON(ctx, "/v1/meetings/templates", &out)
	return out, err
}

// CreateProtocolTemplate saves a protocol template.
func (c *Client) CreateProtocolTemplate(ctx context.Context, name string, schema map[string]any) (Object, error) {
	var out Object
	body := map[string]any{"name": name}
	if schema != nil {
		body["schema"] = schema
	}
	err := c.postJSON(ctx, "/v1/meetings/templates", body, &out)
	return out, err
}

// UpdateProtocolTemplate renames or reshapes a saved template.
func (c *Client) UpdateProtocolTemplate(ctx context.Context, templateID, name string, schema map[string]any) (Object, error) {
	var out Object
	body := map[string]any{"name": name}
	if schema != nil {
		body["schema"] = schema
	}
	err := c.requestJSON(ctx, http.MethodPut, "/v1/meetings/templates/"+url.PathEscape(templateID), nil, body, &out)
	return out, err
}

// DeleteProtocolTemplate removes a saved template.
func (c *Client) DeleteProtocolTemplate(ctx context.Context, templateID string) error {
	_, err := c.send(ctx, http.MethodDelete, "/v1/meetings/templates/"+url.PathEscape(templateID), nil, "", nil)
	return err
}

// WebhookSecret returns the signing secret behind X-VoiceKit-Signature.
func (c *Client) WebhookSecret(ctx context.Context) (Object, error) {
	var out Object
	err := c.getJSON(ctx, "/v1/webhooks/secret", &out)
	return out, err
}

// RotateWebhookSecret issues a new signing secret.
func (c *Client) RotateWebhookSecret(ctx context.Context) (Object, error) {
	var out Object
	err := c.postJSON(ctx, "/v1/webhooks/secret/rotate", map[string]any{}, &out)
	return out, err
}

// Integrations lists external connections.
func (c *Client) Integrations(ctx context.Context) (Object, error) {
	var out Object
	err := c.getJSON(ctx, "/v1/integrations/connections", &out)
	return out, err
}

// ConnectIntegration stores a provider credential.
func (c *Client) ConnectIntegration(ctx context.Context, conn IntegrationConnection) (Object, error) {
	var out Object
	body := compactMap(map[string]any{
		"display_name":  conn.DisplayName,
		"base_url":      conn.BaseURL,
		"refresh_token": conn.RefreshToken,
		"scopes":        conn.Scopes,
	})
	body["provider"] = conn.Provider
	body["secret"] = conn.Secret
	err := c.postJSON(ctx, "/v1/integrations/connections", body, &out)
	return out, err
}

// DeleteIntegration revokes a connection.
func (c *Client) DeleteIntegration(ctx context.Context, connectionID string) error {
	_, err := c.send(ctx, http.MethodDelete, "/v1/integrations/connections/"+url.PathEscape(connectionID), nil, "", nil)
	return err
}

// IntegrationAuthorizeURL starts the OAuth handshake.
func (c *Client) IntegrationAuthorizeURL(ctx context.Context, provider string, consent bool) (Object, error) {
	var out Object
	err := c.postJSON(ctx, "/v1/integrations/"+url.PathEscape(provider)+"/authorize", map[string]any{"consent": consent}, &out)
	return out, err
}

// PushRecording pushes a recording artifact to a CRM deal.
func (c *Client) PushRecording(ctx context.Context, recordingID string, push RecordingPush) (Object, error) {
	var out Object
	kind := push.Kind
	if kind == "" {
		kind = "protocol"
	}
	body := compactMap(map[string]any{
		"deal_id":    push.DealID,
		"phone":      push.Phone,
		"email":      push.Email,
		"source_url": push.SourceURL,
	})
	body["provider"] = push.Provider
	body["kind"] = kind
	err := c.postJSON(ctx, "/v1/recordings/"+url.PathEscape(recordingID)+"/push", body, &out)
	return out, err
}

// RecordingPushes lists the push history of a recording.
func (c *Client) RecordingPushes(ctx context.Context, recordingID string) (Object, error) {
	var out Object
	err := c.getJSON(ctx, "/v1/recordings/"+url.PathEscape(recordingID)+"/pushes", &out)
	return out, err
}

// RetryRecordingPush retries a failed push, reusing its stored target.
func (c *Client) RetryRecordingPush(ctx context.Context, recordingID, provider, kind string) (Object, error) {
	var out Object
	path := "/v1/recordings/" + url.PathEscape(recordingID) + "/pushes/" +
		url.PathEscape(provider) + "/" + url.PathEscape(kind) + "/retry"
	err := c.postJSON(ctx, path, map[string]any{}, &out)
	return out, err
}

// TopUp creates a card top-up payment and returns the payment link.
func (c *Client) TopUp(ctx context.Context, amountRub float64) (Object, error) {
	var out Object
	err := c.postJSON(ctx, "/v1/billing/topup", map[string]any{"amount_rub": amountRub}, &out)
	return out, err
}
