package client

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewWithBaseURL("test-api-key", srv.URL), srv
}

func TestDo_SendsAPIKeyHeader(t *testing.T) {
	t.Setenv("TYPECAST_INTEGRATION_SOURCE", "skill")
	t.Setenv("TYPECAST_GENERATED_BY", "codex")
	var gotKey, gotUserAgent string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-API-KEY")
		gotUserAgent = r.Header.Get("User-Agent")
		w.Write([]byte(`{}`))
	})

	c.get("/test")

	if gotKey != "test-api-key" {
		t.Errorf("expected X-API-KEY = %q, got %q", "test-api-key", gotKey)
	}
	if !strings.HasPrefix(gotUserAgent, "typecast-cli/1.0.7 Go/") ||
		!strings.Contains(gotUserAgent, " net-http (base=custom;") ||
		!strings.HasSuffix(gotUserAgent, " typecast-integration/1 (source=skill; generated_by=codex)") {
		t.Errorf("unexpected User-Agent: %q", gotUserAgent)
	}
}

func TestAttributionSuffix_ValidatesBoundary(t *testing.T) {
	generatedBy := strings.Repeat("a", 32)
	if got := attributionSuffix("skill", generatedBy); !strings.Contains(got, generatedBy) {
		t.Fatalf("expected 32-character generated_by token, got %q", got)
	}
	for _, source := range []string{"api-page", "api-docs"} {
		if got := attributionSuffix(source, "codex"); !strings.Contains(got, "source="+source) {
			t.Fatalf("expected %s attribution, got %q", source, got)
		}
	}
	for _, got := range []string{
		attributionSuffix("other", "codex"),
		attributionSuffix("skill", "Codex"),
		attributionSuffix("skill", generatedBy+"a"),
	} {
		if got != "" {
			t.Fatalf("expected invalid attribution to be omitted, got %q", got)
		}
	}
}

func TestDo_ReturnsErrorOn4xx(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"unauthorized"}`))
	})

	_, err := c.get("/test")
	if err == nil {
		t.Fatal("expected error for 401 response, got nil")
	}
}

func TestDo_ReturnsErrorOn5xx(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"internal server error"}`))
	})

	_, err := c.get("/test")
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}

func TestDo_TextNotSynthesizableIsNotRetried(t *testing.T) {
	requests := 0
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"error_code":"TEXT_NOT_SYNTHESIZABLE","message":"The input text contains characters or symbols that cannot be synthesized into speech. Please check your input text."}`))
	})

	_, err := c.get("/test")
	if err == nil {
		t.Fatal("expected error for 422 response, got nil")
	}
	if requests != 1 {
		t.Fatalf("expected one request, got %d", requests)
	}
	want := "unprocessable request: The input text contains characters or symbols that cannot be synthesized"
	if got := err.Error(); !strings.Contains(got, want) {
		t.Fatalf("expected actionable 422 message, got %q", got)
	}
}

func TestDo_ReturnsBodyOn2xx(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":true}`))
	})

	body, err := c.get("/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestPost_SetsContentTypeHeader(t *testing.T) {
	var gotContentType string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		w.Write([]byte(`{}`))
	})

	c.post("/test", map[string]string{"key": "value"})

	if gotContentType != "application/json" {
		t.Errorf("expected Content-Type = application/json, got %q", gotContentType)
	}
}
