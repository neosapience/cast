package cmd

import (
	"os"
	"testing"

	"github.com/spf13/viper"
)

func TestConfigKeys_ContainsExpectedKeys(t *testing.T) {
	expected := []string{
		"voice-id", "model", "language", "emotion", "emotion-preset", "emotion-intensity",
		"volume", "pitch", "tempo", "format",
	}
	for _, k := range expected {
		if _, ok := configKeys[k]; !ok {
			t.Errorf("configKeys missing expected key %q", k)
		}
	}
}

func TestConfigKeys_MapsToCorrectYAMLKey(t *testing.T) {
	cases := map[string]string{
		"voice-id": "voice_id",
	}
	for flag, yaml := range cases {
		if got := configKeys[flag]; got != yaml {
			t.Errorf("configKeys[%q] = %q, want %q", flag, got, yaml)
		}
	}
}

func TestAvailableKeys_ReturnsAllKeys(t *testing.T) {
	keys := availableKeys()
	if len(keys) != len(configKeys) {
		t.Errorf("availableKeys() returned %d keys, want %d", len(keys), len(configKeys))
	}
}

func TestReadWriteConfig_Roundtrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	want := map[string]any{
		"voice_id": "test-voice",
		"model":    "ssfm-v30",
	}

	if err := writeConfig(want); err != nil {
		t.Fatalf("writeConfig failed: %v", err)
	}

	got, err := readConfig()
	if err != nil {
		t.Fatalf("readConfig failed: %v", err)
	}

	if got["voice_id"] != want["voice_id"] {
		t.Errorf("voice_id = %v, want %v", got["voice_id"], want["voice_id"])
	}
	if got["model"] != want["model"] {
		t.Errorf("model = %v, want %v", got["model"], want["model"])
	}
}

func TestReadConfig_ReturnsEmptyMapIfFileNotExist(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	config, err := readConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(config) != 0 {
		t.Errorf("expected empty config, got %v", config)
	}
}

func TestWriteConfig_SetsRestrictivePermissions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := writeConfig(map[string]any{"key": "value"}); err != nil {
		t.Fatalf("writeConfig failed: %v", err)
	}

	path, _ := configPath()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("expected file permission 0600, got %04o", perm)
	}
}

// newTestViper replicates initConfig() logic on a fresh viper instance
// so tests don't disturb the global viper state (which holds pflag bindings).
func newTestViper(t *testing.T) *viper.Viper {
	t.Helper()
	v := viper.New()
	v.SetEnvPrefix("TYPECAST")
	v.AutomaticEnv()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	v.AddConfigPath(home + "/.typecast")
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.ReadInConfig()
	return v
}

func TestViper_EnvVarOverridesConfigFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("TYPECAST_MODEL", "ssfm-v21")

	if err := writeConfig(map[string]any{"model": "ssfm-v30"}); err != nil {
		t.Fatalf("writeConfig failed: %v", err)
	}

	v := newTestViper(t)
	if got := v.GetString("model"); got != "ssfm-v21" {
		t.Errorf("expected env var (ssfm-v21) to win over config file (ssfm-v30), got %q", got)
	}
}

func TestViper_ConfigFileUsedWhenNoEnvVar(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	os.Unsetenv("TYPECAST_MODEL")
	t.Cleanup(func() { os.Unsetenv("TYPECAST_MODEL") })

	if err := writeConfig(map[string]any{"model": "ssfm-v21"}); err != nil {
		t.Fatalf("writeConfig failed: %v", err)
	}

	v := newTestViper(t)
	if got := v.GetString("model"); got != "ssfm-v21" {
		t.Errorf("expected config file value (ssfm-v21), got %q", got)
	}
}
