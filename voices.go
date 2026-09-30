package voicekit

import (
	"context"
	"errors"
	"net/http"
)

// CreateCloneVoice creates a cloned voice from reference audio (Pro/Business).
// promptText must be the exact transcript of the reference clip; samples may
// contain one or more reference files.
func (c *Client) CreateCloneVoice(ctx context.Context, name, promptText string, samples []Attachment, language string) (Object, error) {
	if len(samples) == 0 {
		return nil, errors.New("voicekit: at least one reference sample is required")
	}
	files := make([]fileField, 0, len(samples))
	for _, sample := range samples {
		files = append(files, fileField{"samples", sample})
	}
	fields := map[string]string{"name": name, "prompt_text": promptText}
	if language != "" {
		fields["language"] = language
	}
	var out Object
	err := c.postMultipart(ctx, "/v1/voices/clone", files, fields, nil, &out)
	return out, err
}

// ListCloneVoices lists the caller's cloned voices.
func (c *Client) ListCloneVoices(ctx context.Context) ([]Object, error) {
	data, err := c.send(ctx, http.MethodGet, "/v1/voices/clone", nil, "", nil)
	if err != nil {
		return nil, err
	}
	return decodeList(data)
}

// GetCloneVoice returns one cloned voice by id.
func (c *Client) GetCloneVoice(ctx context.Context, cloneID string) (Object, error) {
	var out Object
	err := c.getJSON(ctx, "/v1/voices/clone/"+cloneID, &out)
	return out, err
}

// DeleteCloneVoice deletes a cloned voice and its reference latents.
func (c *Client) DeleteCloneVoice(ctx context.Context, cloneID string) error {
	_, err := c.send(ctx, http.MethodDelete, "/v1/voices/clone/"+cloneID, nil, "", nil)
	return err
}

// ───────────────────────────── voice id ───────────────────────────────────

// AnalyzeVoice returns a voice passport: language, gender, age group,
// emotional background, a speaker embedding and an AI-vs-human probability
// (ai_probability is null when anti-spoofing is not configured).
func (c *Client) AnalyzeVoice(ctx context.Context, audio Attachment) (Object, error) {
	var out Object
	err := c.postMultipart(ctx, "/v1/voice-id", []fileField{{"audio", audio}}, nil, nil, &out)
	return out, err
}

// EnrollVoice stores an audio clip as a reusable voice profile; the response
// carries profile_id.
func (c *Client) EnrollVoice(ctx context.Context, audio Attachment, name string) (Object, error) {
	fields := map[string]string{}
	if name != "" {
		fields["name"] = name
	}
	var out Object
	err := c.postMultipart(ctx, "/v1/voice-id/enroll", []fileField{{"audio", audio}}, fields, nil, &out)
	return out, err
}

// VerifyVoice compares a clip against an enrolled profile (1:1) and returns
// profile_id, similarity, verified and threshold.
func (c *Client) VerifyVoice(ctx context.Context, audio Attachment, profileID string) (Object, error) {
	var out Object
	err := c.postMultipart(ctx, "/v1/voice-id/verify", []fileField{{"audio", audio}},
		map[string]string{"profile_id": profileID}, nil, &out)
	return out, err
}

// IdentifyVoice finds the closest matching profile (1:N). Pass nil profileIDs
// to search every profile of the caller.
func (c *Client) IdentifyVoice(ctx context.Context, audio Attachment, profileIDs []string) (Object, error) {
	fields := map[string]string{}
	if ids := joinCSV(profileIDs); ids != "" {
		fields["profile_ids"] = ids
	}
	var out Object
	err := c.postMultipart(ctx, "/v1/voice-id/identify", []fileField{{"audio", audio}}, fields, nil, &out)
	return out, err
}

// ListVoiceProfiles lists the caller's enrolled voice profiles.
func (c *Client) ListVoiceProfiles(ctx context.Context) ([]Object, error) {
	data, err := c.send(ctx, http.MethodGet, "/v1/voice-id/profiles", nil, "", nil)
	if err != nil {
		return nil, err
	}
	return decodeList(data)
}

// DeleteVoiceProfile deletes an enrolled voice profile by id.
func (c *Client) DeleteVoiceProfile(ctx context.Context, profileID string) error {
	_, err := c.send(ctx, http.MethodDelete, "/v1/voice-id/profiles/"+profileID, nil, "", nil)
	return err
}
