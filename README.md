# VoiceKit — Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/lomshakov/voicekit-go.svg)](https://pkg.go.dev/github.com/lomshakov/voicekit-go)
[![CI](https://github.com/lomshakov/voicekit-go/actions/workflows/ci.yml/badge.svg)](https://github.com/lomshakov/voicekit-go/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue)](./LICENSE)

Official Go wrapper for **[VoiceKit](https://ttsapi.ru)** — the REST API for Russian speech:
neural speech synthesis (TTS), transcription (STT) with diarization and timestamps,
sentiment analysis, voice cloning, voice biometrics, audio/video effects, call QA,
semantic search over recordings and batch operations.

> **Links:** [Website](https://ttsapi.ru) · [Documentation](https://ttsapi.ru/docs) · [API reference](https://ttsapi.ru/swagger) · [Pricing](https://ttsapi.ru/pricing) · [Blog](https://ttsapi.ru/blog)

One dependency (`github.com/coder/websocket`, used only by the streaming client);
everything else is the standard library. Requires **Go 1.22+**.

## Install

```bash
go get github.com/lomshakov/voicekit-go
```

## Quick start

```go
package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/lomshakov/voicekit-go"
)

func main() {
	client, err := voicekit.NewClient("rtt_…")
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Synthesis → raw audio bytes
	audio, err := client.Synthesize(ctx, "Привет! Это синтез русской речи.", &voicekit.SynthesizeOptions{
		Voice:  "preset_anna",
		Format: "mp3",
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("speech.mp3", audio, 0o644); err != nil {
		log.Fatal(err)
	}

	// Transcription (async → poll)
	job, err := client.Transcribe(ctx, voicekit.Bytes(audio), &voicekit.TranscribeOptions{
		Language: "ru",
		Keyterms: []string{"диагноз", "препарат"},
	})
	if err != nil {
		log.Fatal(err)
	}
	for {
		result, err := client.GetTranscriptionJob(ctx, job.Str("job_id"))
		if err != nil {
			log.Fatal(err)
		}
		status := result.Str("status")
		if status == "completed" || status == "failed" {
			log.Println(status, result.Str("text"))
			break
		}
		time.Sleep(time.Second)
	}
}
```

### Configuration

```go
client, err := voicekit.NewClient(os.Getenv("VOICEKIT_API_KEY"),
	voicekit.WithBaseURL("https://ttsapi.ru"), // default
	voicekit.WithTimeout(2*time.Minute),       // default 120s
	voicekit.WithHTTPClient(myHTTPClient),     // custom transport / proxy
)
```

### Error handling

Every non-2xx response becomes an `*voicekit.APIError` carrying the RFC 7807
`code`:

```go
_, err := client.SynthesizeStream(ctx, text, nil)
switch {
case errors.Is(err, voicekit.ErrForbidden):
	log.Println("upgrade your plan:", voicekit.ErrorCode(err)) // streaming_forbidden
case voicekit.IsCode(err, "quota_exceeded"):
	log.Println("monthly quota is over")
case errors.Is(err, voicekit.ErrRateLimited):
	log.Println("slow down")
default:
	log.Println(err)
}
```

## Features

| Area | Methods |
| --- | --- |
| Synthesis | `Synthesize`, `SynthesizeStream`, `SynthesizeAsync`, `GetSynthesisJob`, `DownloadSynthesisAudio` |
| Voices | `Voices`, `Voice`, `CreateCloneVoice`, `ListCloneVoices`, `GetCloneVoice`, `DeleteCloneVoice` |
| Transcription | `Transcribe`, `TranscribeSync`, `GetTranscriptionJob`, `Subtitles`, `TranslateTranscript`, `VAD` |
| Analysis | `Analyze`, `AnalyzeSync`, `GetAnalysisJob`, `DetectLanguage`, `Redact`, `Topics`, `Summarize`, `Moderate`, `Evaluate` |
| Effects | `ApplyAudioEffects`, `ApplyVideoEffects`, `CleanAudio` + job/status/download helpers |
| Voice ID | `AnalyzeVoice`, `EnrollVoice`, `VerifyVoice`, `IdentifyVoice`, `ListVoiceProfiles`, `DeleteVoiceProfile` |
| Recordings | `ListRecordings`, `GetRecording`, `RecordingTranscript`, `RecordingSpeakers`, `UpdateRecording`, `UpdateSpeaker`, `RecordingFromLink`, `ExportRecording`, `DownloadRecordingAudio`, share links |
| Call QA | `QAEvaluate`, `QAAnalytics`, `QAEvaluations`, `QAExport` |
| Search | `Search`, `Ask`, `MeetingProtocol` |
| Batch / account | `BatchSynthesize`, `BatchAnalyze`, `GetBatch`, `Usage`, `BillingBalance` |
| Streaming | `TranscribeStream`, `VADStream` (WebSocket, Pro/Business) |

### Streaming synthesis (Pro/Business)

```go
rc, err := client.SynthesizeStream(ctx, "Первое предложение. Второе предложение.", nil)
if err != nil {
	return err
}
defer rc.Close()

out, err := os.Create("long.mp3")
if err != nil {
	return err
}
defer out.Close()
_, err = io.Copy(out, rc) // chunks arrive as the engine produces them
```

### Audio effects (Free/Basic/Pro/Business)

```go
chain, err := voicekit.EffectsJSON([]map[string]any{
	{"type": "reverb", "room_size": 0.5},
	{"type": "pitch", "semitones": 2},
})
if err != nil {
	return err
}

// Inline during synthesis
audio, err := client.Synthesize(ctx, "Привет!", &voicekit.SynthesizeOptions{
	Voice:   "preset_anna",
	Effects: chain,
})

// Or as a background job over an existing file
job, err := client.ApplyAudioEffects(ctx, voicekit.Bytes(audio), &voicekit.AudioEffectsOptions{
	Effects:      []map[string]any{{"type": "compressor", "ratio": 3}},
	OutputFormat: "mp3",
})
```

### Audio cleaning (Free/Basic/Pro/Business)

```go
job, err := client.CleanAudio(ctx, voicekit.Bytes(noisy), nil) // one-click: denoise + normalize
// job, err = client.CleanAudio(ctx, voicekit.Bytes(noisy), &voicekit.CleanAudioOptions{
//	Options: map[string]any{"denoise": map[string]any{"strength": 0.8}, "high_pass": 80},
// })
result, err := client.GetAudioCleaningJob(ctx, job.Str("job_id"))
clean, err := client.DownloadAudioCleaning(ctx, job.Str("job_id"))
```

### Voice ID (Pro/Business)

```go
// Voice passport: language, gender, age, emotion, speaker embedding, AI-vs-human
passport, err := client.AnalyzeVoice(ctx, voicekit.Bytes(sample))

// Voice biometrics over your own profiles
profile, err := client.EnrollVoice(ctx, voicekit.Bytes(speaker), "Alice")
check, err := client.VerifyVoice(ctx, voicekit.Bytes(other), profile.Str("profile_id"))
match, err := client.IdentifyVoice(ctx, voicekit.Bytes(other), nil) // 1:N
```

### Recordings, QA & meeting intelligence (Pro/Business)

```go
recordings, err := client.ListRecordings(ctx, &voicekit.RecordingListOptions{Limit: 10})
job, err := client.RecordingFromLink(ctx, "https://example.com/call.mp3", "ru")
speakers, err := client.RecordingSpeakers(ctx, recordingID)
_, err = client.UpdateSpeaker(ctx, recordingID, "SPEAKER_00", "Иван", "operator")

folder := "Q3"
_, err = client.UpdateRecording(ctx, recordingID, voicekit.UpdateRecordingOptions{
	Tags:   []string{"sales", "warm"},
	Folder: &folder,
})
pdf, err := client.ExportRecording(ctx, recordingID, "pdf")

qa, err := client.QAEvaluate(ctx, recordingID, &voicekit.QAEvaluateOptions{
	Checklist: []map[string]any{{"id": "greeting", "kind": "required", "description": "Поздоровался"}},
})
trend, err := client.QAAnalytics(ctx, 30)

hits, err := client.Search(ctx, "почему клиент отказался?", &voicekit.SearchOptions{Limit: 5, Keywords: "дорого"})
answer, err := client.Ask(ctx, "почему клиент отказался от Pro?")
protocol, err := client.MeetingProtocol(ctx, recordingID, "standup")
```

### WebSocket streaming (Pro/Business)

```go
stream, err := client.TranscribeStream(ctx, &voicekit.StreamOptions{Language: "ru"})
if err != nil {
	return err
}
defer stream.Close()

if err := stream.SendAudio(ctx, pcm16Chunk); err != nil { // raw PCM16, 16 kHz mono
	return err
}
if err := stream.Stop(ctx); err != nil { // finalize the utterance
	return err
}
for {
	event, err := stream.Receive(ctx)
	if errors.Is(err, voicekit.ErrStreamClosed) {
		break
	}
	if err != nil {
		return err
	}
	fmt.Println(event.Str("type"), event.Str("text")) // session / vad / partial / final / error
}
```

### Batches

```go
batch, err := client.BatchSynthesize(ctx, []map[string]any{
	{"text": "Первый текст", "voice": "preset_anna"},
	{"text": "Второй текст", "voice": "preset_dmitri"},
})
status, err := client.GetBatch(ctx, batch.Str("batch_id"))

// Analysis batches take inline base64 audio
audio, err := voicekit.B64File("call.wav")
batch, err = client.BatchAnalyze(ctx, []map[string]any{{"audio": audio, "language": "ru"}})
```

## Dynamic responses

The API evolves, so the SDK returns `voicekit.Object` (`map[string]any`) with
typed accessors instead of freezing every field into a struct:

```go
result, err := client.GetTranscriptionJob(ctx, jobID)
text := result.Str("text")
seconds := result.Num("duration_seconds")
segments := result.List("segments")
for _, segment := range segments {
	fmt.Println(segment.Num("start"), segment.Str("text"))
}
```

## Development

```bash
go vet ./...
go test -race ./...
```

## License

[MIT](./LICENSE) © VoiceKit
