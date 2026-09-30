// Command example walks through the VoiceKit Go SDK.
//
//	VOICEKIT_API_KEY=rtt_… go run ./examples
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lomshakov/voicekit-go"
)

func main() {
	apiKey := os.Getenv("VOICEKIT_API_KEY")
	if apiKey == "" {
		log.Fatal("set VOICEKIT_API_KEY (get a key at https://ttsapi.ru)")
	}

	client, err := voicekit.NewClient(apiKey)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// 1. Synthesize speech.
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
	fmt.Printf("synthesized %d bytes → speech.mp3\n", len(audio))

	// 2. Streaming synthesis (Pro/Business).
	if stream, err := client.SynthesizeStream(ctx, "Первое предложение. Второе предложение.", nil); err == nil {
		_ = stream.Close() // io.Copy(out, stream) for a real consumer
	} else if errors.Is(err, voicekit.ErrForbidden) {
		fmt.Println("streaming needs a Pro or Business plan")
	} else if err != nil {
		log.Fatal(err)
	}

	// 3. Transcribe the produced audio (async job).
	job, err := client.Transcribe(ctx, voicekit.Bytes(audio), &voicekit.TranscribeOptions{Language: "ru"})
	if err != nil {
		log.Fatal(err)
	}
	jobID := job.Str("job_id")
	fmt.Println("transcription queued:", jobID)

	for attempt := 0; attempt < 30; attempt++ {
		result, err := client.GetTranscriptionJob(ctx, jobID)
		if err != nil {
			log.Fatal(err)
		}
		status := result.Str("status")
		if status == "completed" || status == "failed" {
			fmt.Println("transcription:", status, result.Str("text"))
			break
		}
		time.Sleep(time.Second)
	}

	// 4. Text intelligence.
	if language, err := client.DetectLanguage(ctx, "Как дела?"); err == nil {
		fmt.Println("language:", language.Str("language"))
	}

	// 5. Voice catalog and usage.
	if voices, err := client.Voices(ctx); err == nil {
		fmt.Printf("voices available: %d\n", len(voices))
	}
	if usage, err := client.Usage(ctx); err == nil {
		fmt.Println("characters used:", usage.Int("characters_used"))
	}
}
