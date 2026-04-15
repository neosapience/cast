package client

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestTextToSpeech_SendsCorrectRequest(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/text-to-speech" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var body TTSRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.VoiceID != "voice-1" {
			t.Errorf("expected voice_id=voice-1, got %q", body.VoiceID)
		}
		if body.Text != "hello" {
			t.Errorf("expected text=hello, got %q", body.Text)
		}

		w.Write([]byte("audio-bytes"))
	})

	audio, err := c.TextToSpeech(TTSRequest{VoiceID: "voice-1", Text: "hello", Model: "ssfm-v30"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(audio) != "audio-bytes" {
		t.Errorf("unexpected audio response: %s", audio)
	}
}

func TestTextToSpeech_ReturnsErrorOnAPIFailure(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message":"invalid voice_id"}`))
	})

	_, err := c.TextToSpeech(TTSRequest{VoiceID: "bad-id", Text: "hello"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestTextToSpeech_SendsOptionalFields(t *testing.T) {
	seed := 42
	volume := 150
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body TTSRequest
		json.NewDecoder(r.Body).Decode(&body)

		if body.Prompt == nil {
			t.Fatal("expected prompt to be set")
		}
		if body.Prompt.EmotionType != "smart" {
			t.Errorf("expected emotion=smart, got %q", body.Prompt.EmotionType)
		}
		if body.Output == nil {
			t.Fatal("expected output to be set")
		}
		if body.Output.Volume == nil || *body.Output.Volume != 150 {
			t.Errorf("expected volume=150, got %v", body.Output.Volume)
		}
		if body.Seed == nil || *body.Seed != 42 {
			t.Errorf("expected seed=42, got %v", body.Seed)
		}

		w.Write([]byte("ok"))
	})

	c.TextToSpeech(TTSRequest{
		VoiceID: "v1",
		Text:    "hi",
		Prompt:  &TTSPrompt{EmotionType: "smart"},
		Output:  &TTSOutput{Volume: &volume},
		Seed:    &seed,
	})
}

func TestTextToSpeechStream_Success(t *testing.T) {
	// Server sends a known body in one write; the client should deliver all bytes via onChunk.
	want := bytes.Repeat([]byte("ABCD"), 4096) // 16KB to exercise multiple reads
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/text-to-speech/stream" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var body TTSRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.VoiceID != "voice-1" {
			t.Errorf("expected voice_id=voice-1, got %q", body.VoiceID)
		}

		w.Write(want)
	})

	var got bytes.Buffer
	err := c.TextToSpeechStream(TTSRequest{VoiceID: "voice-1", Text: "hello", Model: "ssfm-v30"}, func(chunk []byte) error {
		got.Write(chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(got.Bytes(), want) {
		t.Errorf("received %d bytes, want %d", got.Len(), len(want))
	}
}

func TestTextToSpeechStream_Error(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"unauthorized"}`))
	})

	err := c.TextToSpeechStream(TTSRequest{VoiceID: "v1", Text: "hello"}, func(chunk []byte) error {
		t.Fatal("onChunk should not be called on error")
		return nil
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("authentication failed")) {
		t.Errorf("expected authentication error, got: %v", err)
	}
}
