package voicekit

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := NewClient("test-key", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

func TestNewClient_requires_api_key(t *testing.T) {
	if _, err := NewClient("  "); err == nil {
		t.Fatal("expected an error for an empty API key")
	}
}

func TestSynthesize_sends_key_and_returns_audio(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/synthesize" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("X-Api-Key"); got != "test-key" {
			t.Errorf("X-Api-Key = %q", got)
		}
		body := map[string]any{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body["text"] != "Привет" || body["voice"] != "preset_anna" || body["format"] != "mp3" {
			t.Errorf("unexpected body: %#v", body)
		}
		if _, ok := body["speed"]; ok {
			t.Error("zero speed must be omitted")
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("ID3fake"))
	})

	audio, err := client.Synthesize(context.Background(), "Привет", &SynthesizeOptions{
		Voice:  "preset_anna",
		Format: "mp3",
	})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	if string(audio) != "ID3fake" {
		t.Fatalf("audio = %q", audio)
	}
}

func TestSynthesize_maps_problem_details(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"code":"streaming_forbidden","detail":"Streaming is not included in the 'Free' plan."}`))
	})

	_, err := client.Synthesize(context.Background(), "Привет", nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	if StatusCode(err) != http.StatusForbidden {
		t.Errorf("StatusCode = %d", StatusCode(err))
	}
	if ErrorCode(err) != "streaming_forbidden" {
		t.Errorf("ErrorCode = %q", ErrorCode(err))
	}
	if !errors.Is(err, ErrForbidden) {
		t.Error("errors.Is(err, ErrForbidden) = false")
	}
	if !IsCode(err, "streaming_forbidden") {
		t.Error("IsCode = false")
	}
}

func TestTranscribe_uploads_multipart(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || !strings.HasPrefix(mediaType, "multipart/form-data") {
			t.Fatalf("content type = %q", r.Header.Get("Content-Type"))
		}
		reader := multipart.NewReader(r.Body, params["boundary"])
		form, err := reader.ReadForm(1 << 20)
		if err != nil {
			t.Fatalf("ReadForm: %v", err)
		}
		if got := form.Value["language"]; len(got) != 1 || got[0] != "ru" {
			t.Errorf("language = %#v", got)
		}
		if got := form.Value["keyterms"]; len(got) != 1 || got[0] != "диагноз,препарат" {
			t.Errorf("keyterms = %#v", got)
		}
		if got := form.Value["diarization"]; len(got) != 1 || got[0] != "true" {
			t.Errorf("diarization = %#v", got)
		}
		files := form.File["audio"]
		if len(files) != 1 {
			t.Fatalf("audio parts = %d", len(files))
		}
		file, err := files[0].Open()
		if err != nil {
			t.Fatalf("open part: %v", err)
		}
		defer file.Close()
		data, _ := io.ReadAll(file)
		if string(data) != "RIFFfake" {
			t.Errorf("audio bytes = %q", data)
		}
		_, _ = w.Write([]byte(`{"job_id":"job_1","status":"queued"}`))
	})

	job, err := client.Transcribe(context.Background(), Bytes([]byte("RIFFfake")), &TranscribeOptions{
		Language:    "ru",
		Diarization: true,
		Keyterms:    []string{"диагноз", " препарат "},
	})
	if err != nil {
		t.Fatalf("Transcribe: %v", err)
	}
	if job.Str("job_id") != "job_1" {
		t.Fatalf("job = %#v", job)
	}
}

func TestVoices_decodes_bare_array(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"id":"preset_anna","name":"Anna"}]`))
	})

	voices, err := client.Voices(context.Background())
	if err != nil {
		t.Fatalf("Voices: %v", err)
	}
	if len(voices) != 1 || voices[0].Str("id") != "preset_anna" {
		t.Fatalf("voices = %#v", voices)
	}
}

func TestRecordingTags_decodes_values_wrapper(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"values":["sales","warm"]}`))
	})

	tags, err := client.RecordingTags(context.Background())
	if err != nil {
		t.Fatalf("RecordingTags: %v", err)
	}
	if len(tags) != 2 || tags[0] != "sales" {
		t.Fatalf("tags = %#v", tags)
	}
}

func TestSearch_sends_optional_filters(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body := Object{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body.Str("query") != "почему отказался" || body.Int("limit") != 5 {
			t.Errorf("body = %#v", body)
		}
		if _, ok := body["min_duration_seconds"]; ok {
			t.Error("zero min duration must be omitted")
		}
		_, _ = w.Write([]byte(`{"hits":[]}`))
	})

	result, err := client.Search(context.Background(), "почему отказался", &SearchOptions{Limit: 5, Keywords: "дорого"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if result == nil {
		t.Fatal("expected a response object")
	}
}

func TestEffectsJSON_and_B64(t *testing.T) {
	encoded, err := EffectsJSON([]map[string]any{{"type": "reverb", "room_size": 0.5}})
	if err != nil {
		t.Fatalf("EffectsJSON: %v", err)
	}
	if !strings.Contains(encoded, `"reverb"`) {
		t.Fatalf("encoded = %q", encoded)
	}
	if B64([]byte("abc")) != "YWJj" {
		t.Fatalf("B64 = %q", B64([]byte("abc")))
	}
}
