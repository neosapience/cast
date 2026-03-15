package cmd

import (
	"strings"
	"testing"
)


// resetFlags resets cobra flag values to their defaults between tests.
// Cobra's pflag state is global, so values persist across Execute() calls.
func resetFlags() {
	f := rootCmd.Flags()
	if flag := f.Lookup("volume"); flag != nil {
		flag.Value.Set("-1")
		flag.Changed = false
	}
	if flag := f.Lookup("pitch"); flag != nil {
		flag.Value.Set("0")
		flag.Changed = false
	}
	if flag := f.Lookup("tempo"); flag != nil {
		flag.Value.Set("0")
		flag.Changed = false
	}
	f.Set("emotion", "")
	f.Set("emotion-preset", "")
	if flag := f.Lookup("emotion-intensity"); flag != nil {
		flag.Value.Set("-1")
		flag.Changed = false
	}
	f.Set("prev-text", "")
	f.Set("next-text", "")
	f.Set("model", defaultModel)
	if flag := f.Lookup("seed"); flag != nil {
		flag.Value.Set("-1")
		flag.Changed = false
	}
}

func TestRootCmd_TextValidation(t *testing.T) {
	cases := []struct {
		name   string
		text   string
		errMsg string
	}{
		{"empty text", "", "text must be between 1 and 2000 characters"},
		{"text too long", strings.Repeat("a", 2001), "text must be between 1 and 2000 characters"},
		{"korean text too long", strings.Repeat("가", 2001), "text must be between 1 and 2000 characters"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resetFlags()
			rootCmd.SetArgs([]string{tc.text})
			err := rootCmd.Execute()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.errMsg) {
				t.Errorf("want error %q, got %q", tc.errMsg, err.Error())
			}
		})
	}
}

func TestRootCmd_AudioParamValidation(t *testing.T) {
	cases := []struct {
		name   string
		args   []string
		errMsg string
	}{
		{
			name:   "volume too high",
			args:   []string{"hello", "--volume", "201"},
			errMsg: "volume must be between",
		},
		{
			name:   "pitch too high",
			args:   []string{"hello", "--pitch", "13"},
			errMsg: "pitch must be between",
		},
		{
			name:   "tempo too high",
			args:   []string{"hello", "--tempo", "2.5"},
			errMsg: "tempo must be between",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resetFlags()
			rootCmd.SetArgs(tc.args)
			err := rootCmd.Execute()
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.errMsg) {
				t.Errorf("want error containing %q, got %q", tc.errMsg, err.Error())
			}
		})
	}
}

func TestRootCmd_EmotionValidation(t *testing.T) {
	cases := []struct {
		name   string
		args   []string
		errMsg string
	}{
		{
			name:   "smart requires ssfm-v30",
			args:   []string{"hello", "--emotion", "smart", "--model", "ssfm-v21"},
			errMsg: "emotion 'smart' is only supported with ssfm-v30",
		},
		{
			name:   "prev-text without smart emotion",
			args:   []string{"hello", "--prev-text", "context"},
			errMsg: "--prev-text and --next-text require --emotion smart",
		},
		{
			name:   "next-text without smart emotion",
			args:   []string{"hello", "--next-text", "context", "--emotion", "preset"},
			errMsg: "--prev-text and --next-text require --emotion smart",
		},
		{
			name:   "emotion-preset without emotion preset",
			args:   []string{"hello", "--emotion-preset", "happy"},
			errMsg: "--emotion-preset and --emotion-intensity require --emotion preset",
		},
		{
			name:   "emotion-intensity without emotion preset",
			args:   []string{"hello", "--emotion-intensity", "1.5"},
			errMsg: "--emotion-preset and --emotion-intensity require --emotion preset",
		},
		{
			name:   "unknown emotion type",
			args:   []string{"hello", "--emotion", "invalid"},
			errMsg: "unknown emotion type",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resetFlags()
			rootCmd.SetArgs(tc.args)
			err := rootCmd.Execute()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.errMsg) {
				t.Errorf("want error containing %q, got %q", tc.errMsg, err.Error())
			}
		})
	}
}
