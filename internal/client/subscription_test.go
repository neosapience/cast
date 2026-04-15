package client

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestGetMySubscription_Success(t *testing.T) {
	want := SubscriptionResponse{
		Plan: "plus",
		Credits: Credits{
			PlanCredits: 100000,
			UsedCredits: 1234,
		},
		Limits: Limits{
			ConcurrencyLimit: 5,
		},
	}

	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/users/me/subscription" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		json.NewEncoder(w).Encode(want)
	})

	sub, err := c.GetMySubscription()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sub.Plan != "plus" {
		t.Errorf("expected plan 'plus', got %q", sub.Plan)
	}
	if sub.Credits.PlanCredits != 100000 {
		t.Errorf("expected plan_credits 100000, got %d", sub.Credits.PlanCredits)
	}
	if sub.Credits.UsedCredits != 1234 {
		t.Errorf("expected used_credits 1234, got %d", sub.Credits.UsedCredits)
	}
	if sub.Limits.ConcurrencyLimit != 5 {
		t.Errorf("expected concurrency_limit 5, got %d", sub.Limits.ConcurrencyLimit)
	}
}

func TestGetMySubscription_Unauthorized(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	_, err := c.GetMySubscription()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
