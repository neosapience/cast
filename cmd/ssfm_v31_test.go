package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neosapience/cast/internal/client"
)

func TestSSFMV31SmartRequest(t *testing.T) {
	resetFlags()
	t.Cleanup(resetFlags)
	rootCmd.Flags().Set("model", "ssfm-v31")
	rootCmd.Flags().Set("emotion", "smart")
	rootCmd.Flags().Set("prev-text", "Before")
	req, err := buildTTSRequest(rootCmd, "Hello")
	if err != nil {
		t.Fatal(err)
	}
	if req.Model != "ssfm-v31" || req.Prompt == nil || req.Prompt.EmotionType != "smart" {
		t.Fatalf("unexpected request: %+v", req)
	}
	if defaultModel != "ssfm-v30" {
		t.Fatal("default changed")
	}
}

func TestPickSmartPreviewKeepsSelectedV31Model(t *testing.T) {
	var request client.TTSRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		// Stop before audio playback; only the outgoing request matters here.
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()
	m := newPickModel(testPickVoices(), client.NewWithBaseURL("test", server.URL), "Hello")
	m.presets = []preset{{model: "ssfm-v31"}}
	m.previewSmart()()
	if request.Model != "ssfm-v31" || request.Prompt == nil || request.Prompt.EmotionType != "smart" {
		t.Fatalf("unexpected preview request: %+v", request)
	}
}
