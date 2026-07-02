package client

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestListVoices_ReturnsVoices(t *testing.T) {
	want := []Voice{
		{VoiceID: "v1", VoiceName: "Alice", Gender: "female", Age: "young_adult"},
		{VoiceID: "v2", VoiceName: "Bob", Gender: "male", Age: "middle_age"},
	}

	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/voices" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(want)
	})

	voices, err := c.ListVoices(ListVoicesParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(voices) != 2 {
		t.Fatalf("expected 2 voices, got %d", len(voices))
	}
	if voices[0].VoiceID != "v1" || voices[1].VoiceID != "v2" {
		t.Errorf("unexpected voices: %+v", voices)
	}
}

func TestListVoices_SendsQueryParams(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("gender") != "female" {
			t.Errorf("expected gender=female, got %q", q.Get("gender"))
		}
		if q.Get("age") != "young_adult" {
			t.Errorf("expected age=young_adult, got %q", q.Get("age"))
		}
		json.NewEncoder(w).Encode([]Voice{})
	})

	c.ListVoices(ListVoicesParams{Gender: "female", Age: "young_adult"})
}

func TestListVoices_ReturnsErrorOnAPIFailure(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := c.ListVoices(ListVoicesParams{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetVoice_ReturnsVoice(t *testing.T) {
	want := Voice{VoiceID: "v1", VoiceName: "Alice", Gender: "female"}

	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/voices/v1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(want)
	})

	voice, err := c.GetVoice("v1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if voice.VoiceID != "v1" || voice.VoiceName != "Alice" {
		t.Errorf("unexpected voice: %+v", voice)
	}
}

func TestGetVoice_ReturnsErrorOnAPIFailure(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := c.GetVoice("nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRecommendVoices_ReturnsRecommendations(t *testing.T) {
	want := []RecommendedVoice{
		{VoiceID: "v1", VoiceName: "Alice", Score: 0.92},
		{VoiceID: "v2", VoiceName: "Bob", Score: 0.87},
	}

	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/voices/recommendations" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("query") != "warm narrator" {
			t.Errorf("expected query=warm narrator, got %q", q.Get("query"))
		}
		if q.Get("count") != "5" {
			t.Errorf("expected count=5, got %q", q.Get("count"))
		}
		json.NewEncoder(w).Encode(want)
	})

	voices, err := c.RecommendVoices(RecommendVoicesParams{Query: "warm narrator", Count: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(voices) != 2 {
		t.Fatalf("expected 2 voices, got %d", len(voices))
	}
	if voices[0].VoiceID != "v1" || voices[0].Score != 0.92 {
		t.Errorf("unexpected recommendations: %+v", voices)
	}
}

func TestRecommendVoices_ValidatesInput(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called")
	})

	if _, err := c.RecommendVoices(RecommendVoicesParams{Query: " ", Count: 5}); err == nil {
		t.Fatal("expected empty query error")
	}
	if _, err := c.RecommendVoices(RecommendVoicesParams{Query: "warm", Count: 11}); err == nil {
		t.Fatal("expected count range error")
	}
}

func TestRecommendVoices_DefaultsCountToFive(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("count"); got != "5" {
			t.Errorf("expected default count=5, got %q", got)
		}
		json.NewEncoder(w).Encode([]RecommendedVoice{})
	})

	if _, err := c.RecommendVoices(RecommendVoicesParams{Query: "warm narrator"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
