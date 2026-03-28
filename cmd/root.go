package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/neosapience/cast/internal/client"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:          "cast [text]",
	Short:        "Typecast TTS CLI",
	Args:         cobra.MaximumNArgs(1),
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var text string
		if len(args) == 0 {
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				data, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("failed to read stdin: %w", err)
				}
				text = strings.TrimSpace(string(data))
				if text == "" {
					return fmt.Errorf("no text provided")
				}
			} else {
				return cmd.Help()
			}
		} else {
			text = args[0]
		}

		outFile, _ := cmd.Flags().GetString("out")

		// Infer format from output file extension when format is not set anywhere
		// (flag, env var, or config file).
		if outFile != "" && viper.GetString("format") == "" {
			if ext := strings.ToLower(filepath.Ext(outFile)); ext == ".mp3" || ext == ".wav" {
				viper.Set("format", ext[1:])
			}
		}

		req, err := buildTTSRequest(cmd, text)
		if err != nil {
			return err
		}

		format := viper.GetString("format")

		baseURL := viper.GetString("base_url")
		var c *client.Client
		if baseURL != "" {
			c = client.NewWithBaseURL(viper.GetString("api_key"), baseURL)
		} else {
			c = client.New(viper.GetString("api_key"))
		}

		audio, err := c.TextToSpeech(req)
		if err != nil {
			return err
		}

		if outFile != "" {
			return saveAudio(audio, outFile)
		}
		return playAudio(audio, format)
	},
}

func buildTTSRequest(cmd *cobra.Command, text string) (client.TTSRequest, error) {
	runeCount := utf8.RuneCountInString(text)
	if runeCount == 0 || runeCount > 2000 {
		return client.TTSRequest{}, fmt.Errorf("text must be between 1 and 2000 characters, got %d", runeCount)
	}

	flags := cmd.Flags()
	voiceID := viper.GetString("voice_id")
	model := viper.GetString("model")
	language := viper.GetString("language")
	emotion := viper.GetString("emotion")
	emotionPreset := viper.GetString("emotion_preset")
	emotionIntensity := viper.GetFloat64("emotion_intensity")
	prevText := viper.GetString("prev_text")
	nextText := viper.GetString("next_text")
	volume := viper.GetInt("volume")
	pitch := viper.GetInt("pitch")
	tempo := viper.GetFloat64("tempo")
	format := viper.GetString("format")
	seed := viper.GetInt("seed")

	// Validate all params before producing any side effects (e.g. warning messages).
	if emotion != "" && emotion != "smart" && emotion != "preset" {
		return client.TTSRequest{}, fmt.Errorf("unknown emotion type %q, use 'smart' or 'preset'", emotion)
	}
	if emotion == "smart" && model != "ssfm-v30" {
		return client.TTSRequest{}, fmt.Errorf("emotion 'smart' is only supported with ssfm-v30")
	}
	if (prevText != "" || nextText != "") && emotion != "smart" {
		return client.TTSRequest{}, fmt.Errorf("--prev-text and --next-text require --emotion smart")
	}
	if (emotionPreset != "" || flags.Changed("emotion-intensity")) && emotion != "preset" {
		return client.TTSRequest{}, fmt.Errorf("--emotion-preset and --emotion-intensity require --emotion preset")
	}
	if emotionIntensity >= 0 {
		if emotionIntensity < minEmotionIntensity || emotionIntensity > maxEmotionIntensity {
			return client.TTSRequest{}, fmt.Errorf("emotion-intensity must be between %.1f and %.1f, got %g", minEmotionIntensity, maxEmotionIntensity, emotionIntensity)
		}
	}
	if volume >= 0 {
		if volume < minVolume || volume > maxVolume {
			return client.TTSRequest{}, fmt.Errorf("volume must be between %d and %d, got %d", minVolume, maxVolume, volume)
		}
	}
	// pitch uses 0 as both sentinel and valid value; pitch=0 (no change) is
	// semantically equivalent to omitting it, so this does not cause data loss.
	if flags.Changed("pitch") || pitch != 0 {
		if pitch < minPitch || pitch > maxPitch {
			return client.TTSRequest{}, fmt.Errorf("pitch must be between %d and %d, got %d", minPitch, maxPitch, pitch)
		}
	}
	if tempo >= 0 {
		if tempo < minTempo || tempo > maxTempo {
			return client.TTSRequest{}, fmt.Errorf("tempo must be between %.1f and %.1f, got %g", minTempo, maxTempo, tempo)
		}
	}

	if voiceID == "" {
		voiceID = defaultVoiceID
		fmt.Fprintf(os.Stderr, "voice ID not set, using default: %s\n", defaultVoiceID)
	}

	req := client.TTSRequest{
		VoiceID:  voiceID,
		Text:     text,
		Model:    model,
		Language: language,
	}

	if emotion != "" {
		prompt := &client.TTSPrompt{}
		switch emotion {
		case "smart":
			prompt.EmotionType = "smart"
			prompt.PreviousText = prevText
			prompt.NextText = nextText
		case "preset":
			prompt.EmotionType = "preset"
			prompt.EmotionPreset = emotionPreset
			if emotionIntensity >= 0 {
				prompt.EmotionIntensity = &emotionIntensity
			}
		}
		req.Prompt = prompt
	}

	out := &client.TTSOutput{}
	hasOutput := false
	if volume >= 0 {
		out.Volume = &volume
		hasOutput = true
	}
	// See validation comment above regarding pitch sentinel.
	if flags.Changed("pitch") || pitch != 0 {
		out.AudioPitch = &pitch
		hasOutput = true
	}
	if tempo >= 0 {
		out.AudioTempo = &tempo
		hasOutput = true
	}
	if format != "" {
		out.AudioFormat = format
		hasOutput = true
	}
	if hasOutput {
		req.Output = out
	}

	if seed >= 0 {
		req.Seed = &seed
	}

	return req, nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().String("api-key", "", "API key (or TYPECAST_API_KEY)")
	viper.BindPFlag("api_key", rootCmd.PersistentFlags().Lookup("api-key"))
	rootCmd.PersistentFlags().String("base-url", "", "API base URL (or TYPECAST_BASE_URL, default: https://api.typecast.ai)")
	viper.BindPFlag("base_url", rootCmd.PersistentFlags().Lookup("base-url"))

	f := rootCmd.Flags()
	f.String("voice-id", "", "Voice ID (or TYPECAST_VOICE_ID)")
	viper.BindPFlag("voice_id", f.Lookup("voice-id"))
	f.String("model", defaultModel, "Model (ssfm-v30, ssfm-v21)")
	viper.BindPFlag("model", f.Lookup("model"))
	f.String("language", "", "Language code (ISO 639-3, auto-detected if omitted)")
	viper.BindPFlag("language", f.Lookup("language"))
	f.String("out", "", "Save to file instead of playing")
	f.String("emotion", "", "Emotion type: smart (ssfm-v30 only), preset")
	viper.BindPFlag("emotion", f.Lookup("emotion"))
	f.String("emotion-preset", "", "Emotion preset (normal, happy, sad, angry, whisper, toneup, tonedown — see voices for availability)")
	viper.BindPFlag("emotion_preset", f.Lookup("emotion-preset"))
	f.Float64("emotion-intensity", -1, "Emotion intensity (0.0–2.0, default 1.0)")
	viper.BindPFlag("emotion_intensity", f.Lookup("emotion-intensity"))
	f.String("prev-text", "", "Previous text for context")
	viper.BindPFlag("prev_text", f.Lookup("prev-text"))
	f.String("next-text", "", "Next text for context")
	viper.BindPFlag("next_text", f.Lookup("next-text"))
	f.Int("volume", -1, "Volume (0-200, default 100)")
	viper.BindPFlag("volume", f.Lookup("volume"))
	f.Int("pitch", 0, "Pitch in semitones (-12 to +12, default 0)")
	viper.BindPFlag("pitch", f.Lookup("pitch"))
	f.Float64("tempo", -1, "Tempo multiplier (0.5-2.0, default 1.0)")
	viper.BindPFlag("tempo", f.Lookup("tempo"))
	f.String("format", "", "Output format (wav, mp3)")
	viper.BindPFlag("format", f.Lookup("format"))
	f.Int("seed", -1, "Random seed for reproducible output")
	viper.BindPFlag("seed", f.Lookup("seed"))
}

func initConfig() {
	viper.SetEnvPrefix("TYPECAST")
	viper.AutomaticEnv()

	home, err := os.UserHomeDir()
	if err == nil {
		viper.AddConfigPath(home + "/.typecast")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.ReadInConfig()
	}
}
