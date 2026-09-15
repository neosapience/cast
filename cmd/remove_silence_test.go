package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestRemoveSilenceEnvironment(t *testing.T) {
	viper.SetEnvPrefix("TYPECAST")
	viper.AutomaticEnv()
	resetFlags()
	defer resetFlags()
	t.Setenv("TYPECAST_REMOVE_SILENCE_MS", "0")
	req, err := buildTTSRequest(rootCmd, "test")
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(req)
	if err != nil || !strings.Contains(string(data), `"remove_silence_ms":0`) {
		t.Fatal(string(data), err)
	}
	t.Setenv("TYPECAST_REMOVE_SILENCE_MS", "1.5")
	if _, err := buildTTSRequest(rootCmd, "test"); err == nil {
		t.Fatal("invalid environment value accepted")
	}
}

func TestRemoveSilenceOption(t *testing.T) {
	defer resetFlags()
	resetFlags()
	req, err := buildTTSRequest(rootCmd, "test")
	if err != nil || (req.Output != nil && req.Output.RemoveSilenceMS != nil) {
		t.Fatal(req, err)
	}
	for _, raw := range []string{"0", "300", "1000", "-1", "1001", "true", "1.5", ""} {
		resetFlags()
		if err := rootCmd.Flags().Set("remove-silence-ms", raw); err != nil {
			t.Fatal(err)
		}
		req, err := buildTTSRequest(rootCmd, "test")
		valid := raw == "0" || raw == "300" || raw == "1000"
		if (err == nil) != valid {
			t.Fatal(raw, err)
		}
		if valid && (req.Output == nil || req.Output.RemoveSilenceMS == nil) {
			t.Fatal(raw, req)
		}
	}
}
