package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestBuildTTSRequest_EmotionPreset(t *testing.T) {
	cases := []struct {
		name              string
		model             string
		preset            string
		intensity         string
		wantEmotionType   string
		wantEmotionPreset string
	}{
		{
			name:              "preset on ssfm-v21",
			model:             "ssfm-v21",
			preset:            "happy",
			wantEmotionType:   "preset",
			wantEmotionPreset: "happy",
		},
		{
			name:              "preset on ssfm-v30",
			model:             "ssfm-v30",
			preset:            "whisper",
			wantEmotionType:   "preset",
			wantEmotionPreset: "whisper",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resetFlags()
			rootCmd.Flags().Set("emotion", "preset")
			rootCmd.Flags().Set("emotion-preset", tc.preset)
			rootCmd.Flags().Set("model", tc.model)

			req, err := buildTTSRequest(rootCmd, "hello")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if req.Prompt == nil {
				t.Fatal("expected prompt to be set")
			}
			if req.Prompt.EmotionType != tc.wantEmotionType {
				t.Errorf("emotion_type: want %q, got %q", tc.wantEmotionType, req.Prompt.EmotionType)
			}
			if req.Prompt.EmotionPreset != tc.wantEmotionPreset {
				t.Errorf("emotion_preset: want %q, got %q", tc.wantEmotionPreset, req.Prompt.EmotionPreset)
			}
		})
	}
}

func TestBuildTTSRequest_EmotionIntensity(t *testing.T) {
	resetFlags()
	rootCmd.Flags().Set("emotion", "preset")
	rootCmd.Flags().Set("emotion-preset", "happy")
	rootCmd.Flags().Set("emotion-intensity", "1.5")
	rootCmd.Flags().Set("model", defaultModel)

	req, err := buildTTSRequest(rootCmd, "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Prompt == nil || req.Prompt.EmotionIntensity == nil {
		t.Fatal("expected emotion_intensity to be set")
	}
	if *req.Prompt.EmotionIntensity != 1.5 {
		t.Errorf("emotion_intensity: want 1.5, got %g", *req.Prompt.EmotionIntensity)
	}
}


func TestBuildTTSRequest_AudioParams(t *testing.T) {
	cases := []struct {
		name       string
		volume     string
		pitch      string
		tempo      string
		format     string
		wantVolume *int
		wantPitch  *int
		wantTempo  *float64
		wantFormat string
	}{
		{
			name: "volume and pitch set",
			volume: "150", pitch: "6",
			wantVolume: intPtr(150), wantPitch: intPtr(6),
		},
		{
			name: "tempo set",
			tempo: "1.5",
			wantTempo: float64Ptr(1.5),
		},
		{
			name: "format mp3",
			format: "mp3",
			wantFormat: "mp3",
		},
		{
			name: "negative pitch",
			pitch: "-6",
			wantPitch: intPtr(-6),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resetFlags()
			if tc.volume != "" {
				rootCmd.Flags().Set("volume", tc.volume)
			}
			if tc.pitch != "" {
				rootCmd.Flags().Set("pitch", tc.pitch)
			}
			if tc.tempo != "" {
				rootCmd.Flags().Set("tempo", tc.tempo)
			}
			if tc.format != "" {
				viper.Set("format", tc.format)
			}

			req, err := buildTTSRequest(rootCmd, "hello")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.wantVolume != nil {
				if req.Output == nil || req.Output.Volume == nil || *req.Output.Volume != *tc.wantVolume {
					t.Errorf("volume: want %v, got %v", tc.wantVolume, outputField(req.Output, "volume"))
				}
			}
			if tc.wantPitch != nil {
				if req.Output == nil || req.Output.AudioPitch == nil || *req.Output.AudioPitch != *tc.wantPitch {
					t.Errorf("pitch: want %v, got %v", tc.wantPitch, outputField(req.Output, "pitch"))
				}
			}
			if tc.wantTempo != nil {
				if req.Output == nil || req.Output.AudioTempo == nil || *req.Output.AudioTempo != *tc.wantTempo {
					t.Errorf("tempo: want %v, got %v", tc.wantTempo, outputField(req.Output, "tempo"))
				}
			}
			if tc.wantFormat != "" {
				if req.Output == nil || req.Output.AudioFormat != tc.wantFormat {
					t.Errorf("format: want %q, got %q", tc.wantFormat, outputField(req.Output, "format"))
				}
			}

			viper.Set("format", "")
		})
	}
}

func TestBuildTTSRequest_Seed(t *testing.T) {
	resetFlags()
	rootCmd.Flags().Set("seed", "42")

	req, err := buildTTSRequest(rootCmd, "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Seed == nil || *req.Seed != 42 {
		t.Errorf("seed: want 42, got %v", req.Seed)
	}
}

func TestBuildTTSRequest_Language(t *testing.T) {
	resetFlags()
	viper.Set("language", "eng")
	defer viper.Set("language", "")

	req, err := buildTTSRequest(rootCmd, "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Language != "eng" {
		t.Errorf("language: want %q, got %q", "eng", req.Language)
	}
}

func TestBuildTTSRequest_SmartEmotionContext(t *testing.T) {
	resetFlags()
	rootCmd.Flags().Set("emotion", "smart")
	rootCmd.Flags().Set("model", "ssfm-v30")
	rootCmd.Flags().Set("prev-text", "오늘 복권에 당첨됐어요!")
	rootCmd.Flags().Set("next-text", "믿기지가 않네요.")

	req, err := buildTTSRequest(rootCmd, "정말요?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Prompt == nil {
		t.Fatal("expected prompt to be set")
	}
	if req.Prompt.EmotionType != "smart" {
		t.Errorf("emotion_type: want %q, got %q", "smart", req.Prompt.EmotionType)
	}
	if req.Prompt.PreviousText != "오늘 복권에 당첨됐어요!" {
		t.Errorf("previous_text: want %q, got %q", "오늘 복권에 당첨됐어요!", req.Prompt.PreviousText)
	}
	if req.Prompt.NextText != "믿기지가 않네요." {
		t.Errorf("next_text: want %q, got %q", "믿기지가 않네요.", req.Prompt.NextText)
	}
}

func TestBuildTTSRequest_OutputNilWhenDefaults(t *testing.T) {
	resetFlags()

	req, err := buildTTSRequest(rootCmd, "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Output != nil {
		t.Errorf("expected output to be nil when no audio params set, got %+v", req.Output)
	}
	if req.Prompt != nil {
		t.Errorf("expected prompt to be nil when no emotion set, got %+v", req.Prompt)
	}
	if req.Seed != nil {
		t.Errorf("expected seed to be nil when not set, got %v", req.Seed)
	}
}

func TestRootCmd_OutFileSaved(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		w.Write([]byte("fake-audio-bytes"))
	}))
	defer srv.Close()

	outFile := filepath.Join(t.TempDir(), "out.mp3")
	resetFlags()
	rootCmd.SetArgs([]string{"hello", "--base-url", srv.URL, "--out", outFile})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("output file not found: %v", err)
	}
	if string(data) != "fake-audio-bytes" {
		t.Errorf("file content: want %q, got %q", "fake-audio-bytes", string(data))
	}
}

func TestRootCmd_OutFileFormatInferredFromExtension(t *testing.T) {
	var capturedFormat string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Output *struct {
				AudioFormat string `json:"audio_format"`
			} `json:"output"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if body.Output != nil {
			capturedFormat = body.Output.AudioFormat
		}
		w.Write([]byte("audio"))
	}))
	defer srv.Close()

	for _, ext := range []string{"mp3", "wav"} {
		t.Run(ext, func(t *testing.T) {
			capturedFormat = ""
			outFile := filepath.Join(t.TempDir(), "out."+ext)
			resetFlags()
			rootCmd.SetArgs([]string{"hello", "--base-url", srv.URL, "--out", outFile})
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if capturedFormat != ext {
				t.Errorf("format inferred from .%s: want %q, got %q", ext, ext, capturedFormat)
			}
		})
	}
}

// helpers

func intPtr(v int) *int       { return &v }
func float64Ptr(v float64) *float64 { return &v }

func outputField(out interface{}, field string) string {
	if out == nil {
		return "<nil output>"
	}
	return "<set>"
}

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
		flag.Value.Set("-1")
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
	f.Set("out", "")
	f.Set("format", "")
	f.Set("language", "")
	if flag := f.Lookup("target-lufs"); flag != nil {
		flag.Value.Set("0")
		flag.Changed = false
	}
	viper.Set("format", "")
	viper.Set("language", "")
}

func TestRootCmd_TargetLufsFlag(t *testing.T) {
	resetFlags()
	rootCmd.Flags().Set("target-lufs", "-14")

	req, err := buildTTSRequest(rootCmd, "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Output == nil || req.Output.TargetLUFS == nil {
		t.Fatal("expected target_lufs to be set")
	}
	if *req.Output.TargetLUFS != -14.0 {
		t.Errorf("target_lufs: want -14, got %g", *req.Output.TargetLUFS)
	}
	if req.Output.Volume != nil {
		t.Errorf("volume should be nil when target_lufs is set, got %v", *req.Output.Volume)
	}
}

func TestRootCmd_VolumeAndTargetLufsMutuallyExclusive(t *testing.T) {
	resetFlags()
	rootCmd.Flags().Set("volume", "100")
	rootCmd.Flags().Set("target-lufs", "-14")

	_, err := buildTTSRequest(rootCmd, "hello")
	if err == nil {
		t.Fatal("expected error when both --volume and --target-lufs are set")
	}
	if !strings.Contains(err.Error(), "cannot use both --volume and --target-lufs") {
		t.Errorf("unexpected error: %v", err)
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
