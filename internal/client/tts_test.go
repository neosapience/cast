package client

import (
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

func TestTTSRequest_TargetLUFS_SerializesCorrectly(t *testing.T) {
	lufs := -14.0
	req := TTSRequest{
		VoiceID: "v1",
		Text:    "hello",
		Model:   "ssfm-v30",
		Output:  &TTSOutput{TargetLUFS: &lufs},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var m map[string]interface{}
	json.Unmarshal(data, &m)

	out, ok := m["output"].(map[string]interface{})
	if !ok {
		t.Fatal("expected output in JSON")
	}
	if _, exists := out["target_lufs"]; !exists {
		t.Error("expected target_lufs in JSON output")
	}
	if val := out["target_lufs"].(float64); val != -14.0 {
		t.Errorf("target_lufs: want -14, got %g", val)
	}
	if _, exists := out["volume"]; exists {
		t.Error("volume should not be present when target_lufs is set")
	}
}

func TestTTSRequest_Volume_NoTargetLUFS(t *testing.T) {
	vol := 150
	req := TTSRequest{
		VoiceID: "v1",
		Text:    "hello",
		Model:   "ssfm-v30",
		Output:  &TTSOutput{Volume: &vol},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var m map[string]interface{}
	json.Unmarshal(data, &m)

	out, ok := m["output"].(map[string]interface{})
	if !ok {
		t.Fatal("expected output in JSON")
	}
	if _, exists := out["volume"]; !exists {
		t.Error("expected volume in JSON output")
	}
	if val := out["volume"].(float64); val != 150 {
		t.Errorf("volume: want 150, got %g", val)
	}
	if _, exists := out["target_lufs"]; exists {
		t.Error("target_lufs should not be present when only volume is set")
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
