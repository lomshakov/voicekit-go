package voicekit

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// APIError is returned when the API answers with a non-2xx status. The payload
// follows RFC 7807 Problem Details, so Code is the machine-readable error code
// (e.g. "quota_exceeded", "streaming_forbidden").
type APIError struct {
	// Status is the HTTP status code.
	Status int
	// Code is the stable error code from the response body ("" when absent).
	Code string
	// Message is the human-readable detail (or title).
	Message string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("voicekit: %s (HTTP %d, code %s)", e.Message, e.Status, e.Code)
	}
	return fmt.Sprintf("voicekit: %s (HTTP %d)", e.Message, e.Status)
}

// Is matches sentinel errors by status: a sentinel without a Code matches any
// error carrying the same HTTP status.
func (e *APIError) Is(target error) bool {
	var other *APIError
	if !errors.As(target, &other) {
		return false
	}
	if other.Code == "" {
		return other.Status == e.Status
	}
	return other.Status == e.Status && other.Code == e.Code
}

// StatusCode returns the HTTP status carried by err, or 0 when err is not an
// APIError.
func StatusCode(err error) int {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Status
	}
	return 0
}

// ErrorCode returns the machine-readable code carried by err, or "".
func ErrorCode(err error) string {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Code
	}
	return ""
}

// IsCode reports whether err is an APIError with the given error code.
func IsCode(err error, code string) bool { return ErrorCode(err) == code }

// Sentinel errors for the most common statuses; combine with errors.Is:
//
//	if errors.Is(err, voicekit.ErrUnauthorized) { … }
var (
	// ErrUnauthorized is a 401 response (missing or invalid API key).
	ErrUnauthorized = &APIError{Status: http.StatusUnauthorized}
	// ErrForbidden is a 403 response (plan does not include the feature).
	ErrForbidden = &APIError{Status: http.StatusForbidden}
	// ErrNotFound is a 404 response.
	ErrNotFound = &APIError{Status: http.StatusNotFound}
	// ErrRateLimited is a 429 response.
	ErrRateLimited = &APIError{Status: http.StatusTooManyRequests}
)

func newAPIError(status int, body []byte) *APIError {
	err := &APIError{Status: status}
	var payload struct {
		Code   string `json:"code"`
		Detail string `json:"detail"`
		Title  string `json:"title"`
	}
	if json.Unmarshal(body, &payload) == nil {
		err.Code = payload.Code
		switch {
		case payload.Detail != "":
			err.Message = payload.Detail
		case payload.Title != "":
			err.Message = payload.Title
		}
	}
	if err.Message == "" {
		if text := string(body); len(text) > 0 && len(text) < 512 {
			err.Message = text
		} else {
			err.Message = http.StatusText(status)
		}
	}
	return err
}

func base64Encode(data []byte) string { return base64.StdEncoding.EncodeToString(data) }
