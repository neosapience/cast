package cmd

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestFormatInt(t *testing.T) {
	cases := []struct {
		name string
		n    int64
		want string
	}{
		{"zero", 0, "0"},
		{"small", 123, "123"},
		{"thousands", 1234, "1,234"},
		{"millions", 1234567, "1,234,567"},
		{"negative", -1234, "-1,234"},
		{"negative millions", -1234567, "-1,234,567"},
		{"MinInt64", math.MinInt64, "-9,223,372,036,854,775,808"},
		{"MaxInt64", math.MaxInt64, "9,223,372,036,854,775,807"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatInt(tc.n)
			if got != tc.want {
				t.Errorf("formatInt(%d) = %q, want %q", tc.n, got, tc.want)
			}
		})
	}
}

func TestSubscriptionCmd_JSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/users/me/subscription" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Write([]byte(`{"plan":"plus","credits":{"plan_credits":1000,"used_credits":250},"limits":{"concurrency_limit":5}}`))
	}))
	defer srv.Close()

	resetFlags()
	resetSubscriptionTestConfig()
	viper.Set("base_url", srv.URL)
	t.Cleanup(resetSubscriptionTestConfig)

	var out bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = oldStdout })

	rootCmd.SetArgs([]string{"subscription", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	w.Close()
	out.ReadFrom(r)

	var got struct {
		Plan             string `json:"plan"`
		RemainingCredits int64  `json:"remaining_credits"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, out.String())
	}
	if got.Plan != "plus" {
		t.Errorf("plan: want plus, got %q", got.Plan)
	}
	if got.RemainingCredits != 750 {
		t.Errorf("remaining_credits: want 750, got %d", got.RemainingCredits)
	}
}

func TestSubscriptionCmd_HumanReadable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"plan":"lite","credits":{"plan_credits":2000,"used_credits":1250},"limits":{"concurrency_limit":3}}`))
	}))
	defer srv.Close()

	resetFlags()
	resetSubscriptionTestConfig()
	viper.Set("base_url", srv.URL)
	t.Cleanup(resetSubscriptionTestConfig)

	var out bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = oldStdout })

	rootCmd.SetArgs([]string{"subscription"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	w.Close()
	out.ReadFrom(r)

	if !strings.Contains(out.String(), "Plan:        lite") {
		t.Errorf("unexpected output: %s", out.String())
	}
	if !strings.Contains(out.String(), "Remaining:   750") {
		t.Errorf("unexpected output: %s", out.String())
	}
}

func resetSubscriptionTestConfig() {
	if flag := subscriptionCmd.Flags().Lookup("json"); flag != nil {
		flag.Value.Set("false")
		flag.Changed = false
	}
	viper.Set("base_url", "")
	viper.Set("emotion", "")
	viper.Set("emotion_preset", "")
	viper.Set("prev_text", "")
	viper.Set("next_text", "")
	viper.Set("volume", -1)
	viper.Set("pitch", 0)
	viper.Set("tempo", -1)
	viper.Set("seed", -1)
}
