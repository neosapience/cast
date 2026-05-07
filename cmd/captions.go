package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/neosapience/cast/internal/client"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var captionsCmd = &cobra.Command{
	Use:   "captions [text]",
	Short: "Generate timestamp-aligned captions (SRT/VTT) alongside the audio",
	Long: `Generate timestamp-aligned captions (SRT or WebVTT) for the given text.

Internally calls POST /v1/text-to-speech/with-timestamps and groups the
returned word/character alignment into cues that match the Python, JS, and
Go SDK output byte-for-byte (sentence boundary + 7s/42-char limit).

For non-whitespace languages (jpn, zho), pass --granularity char or both
(or set --language jpn|zho and let cast auto-fallback to char).`,
	Args:          cobra.MaximumNArgs(1),
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		text, err := readCaptionsText(args)
		if err != nil {
			return err
		}

		flags := cmd.Flags()
		captionFormat, _ := flags.GetString("format")
		captionsOut, _ := flags.GetString("captions-out")
		audioOut, _ := flags.GetString("audio-out")
		granularity, _ := flags.GetString("granularity")

		if captionFormat == "" {
			return fmt.Errorf("--format is required (srt or vtt)")
		}
		captionFormat = strings.ToLower(captionFormat)
		if captionFormat != "srt" && captionFormat != "vtt" {
			return fmt.Errorf("--format must be 'srt' or 'vtt', got %q", captionFormat)
		}
		if captionsOut == "" {
			return fmt.Errorf("--captions-out is required")
		}
		if granularity != "" {
			granularity = strings.ToLower(granularity)
			switch granularity {
			case "word", "char", "both":
			default:
				return fmt.Errorf("--granularity must be 'word', 'char', or 'both', got %q", granularity)
			}
		}

		// Promote captions-local flags into viper so buildTTSRequest sees them.
		applyCaptionsFlagsToViper(cmd)

		// Auto-fallback: jpn/zho whitespace-less languages prefer character granularity.
		language := viper.GetString("language")
		if granularity == "" && (language == "jpn" || language == "zho") {
			granularity = "char"
			fmt.Fprintf(os.Stderr, "language %q has no whitespace, defaulting --granularity char\n", language)
		}

		req, err := buildTTSRequest(cmd, text)
		if err != nil {
			return err
		}

		baseURL := viper.GetString("base_url")
		var c *client.Client
		if baseURL != "" {
			c = client.NewWithBaseURL(viper.GetString("api_key"), baseURL)
		} else {
			c = client.New(viper.GetString("api_key"))
		}

		tsReq := client.TTSRequestWithTimestamps(req)

		resp, err := c.TextToSpeechWithTimestamps(tsReq, granularity)
		if err != nil {
			return err
		}

		var caption string
		switch captionFormat {
		case "srt":
			caption, err = resp.ToSRT()
		case "vtt":
			caption, err = resp.ToVTT()
		}
		if err != nil {
			return fmt.Errorf("caption generation failed: %w", err)
		}

		if err := os.WriteFile(captionsOut, []byte(caption), 0644); err != nil {
			return fmt.Errorf("failed to write captions to %s: %w", captionsOut, err)
		}
		fmt.Fprintf(os.Stderr, "captions saved: %s\n", captionsOut)

		if audioOut != "" {
			if err := resp.SaveAudio(audioOut); err != nil {
				return fmt.Errorf("failed to save audio: %w", err)
			}
			fmt.Fprintf(os.Stderr, "audio saved: %s\n", audioOut)
		}
		return nil
	},
}

func readCaptionsText(args []string) (string, error) {
	if len(args) == 1 {
		return args[0], nil
	}
	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to stat stdin: %w", err)
	}
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("failed to read stdin: %w", err)
		}
		text := strings.TrimSpace(string(data))
		if text == "" {
			return "", fmt.Errorf("no text provided")
		}
		return text, nil
	}
	return "", fmt.Errorf("no text provided (pass as argument or pipe via stdin)")
}

// applyCaptionsFlagsToViper promotes captions-local flag values into viper
// so that buildTTSRequest (which reads from viper) sees them. We only set
// keys that the user explicitly changed on the captions command, so env
// variables and config file fallbacks still work for the rest.
func applyCaptionsFlagsToViper(cmd *cobra.Command) {
	flags := cmd.Flags()
	stringKeys := map[string]string{
		"voice-id":       "voice_id",
		"model":          "model",
		"language":       "language",
		"emotion":        "emotion",
		"emotion-preset": "emotion_preset",
		"prev-text":      "prev_text",
		"next-text":      "next_text",
		"audio-format":   "format",
	}
	for flagName, viperKey := range stringKeys {
		if flags.Changed(flagName) {
			if v, err := flags.GetString(flagName); err == nil {
				viper.Set(viperKey, v)
			}
		}
	}
	if flags.Changed("emotion-intensity") {
		if v, err := flags.GetFloat64("emotion-intensity"); err == nil {
			viper.Set("emotion_intensity", v)
		}
	}
	if flags.Changed("volume") {
		if v, err := flags.GetInt("volume"); err == nil {
			viper.Set("volume", v)
		}
	}
	if flags.Changed("pitch") {
		if v, err := flags.GetInt("pitch"); err == nil {
			viper.Set("pitch", v)
		}
	}
	if flags.Changed("tempo") {
		if v, err := flags.GetFloat64("tempo"); err == nil {
			viper.Set("tempo", v)
		}
	}
	if flags.Changed("seed") {
		if v, err := flags.GetInt("seed"); err == nil {
			viper.Set("seed", v)
		}
	}
}

func init() {
	// Caption-specific flags
	captionsCmd.Flags().String("format", "", "Caption format: srt or vtt (required)")
	captionsCmd.Flags().String("captions-out", "", "Caption output file path (required)")
	captionsCmd.Flags().String("audio-out", "", "Optional audio output file path; omit to skip writing audio")
	captionsCmd.Flags().String("granularity", "", "Alignment granularity: word, char, or both (default: server default 'word'; auto 'char' for jpn/zho)")

	// TTS flags mirrored from rootCmd so users can pass them on the captions
	// subcommand. Values are promoted into viper at runtime — see
	// applyCaptionsFlagsToViper. Defaults match rootCmd exactly.
	captionsCmd.Flags().String("voice-id", "", "Voice ID (or TYPECAST_VOICE_ID)")
	captionsCmd.Flags().String("model", defaultModel, "Model (ssfm-v30, ssfm-v21)")
	captionsCmd.Flags().String("language", "", "Language code (ISO 639-3, auto-detected if omitted)")
	captionsCmd.Flags().String("emotion", "", "Emotion type: smart (ssfm-v30 only), preset")
	captionsCmd.Flags().String("emotion-preset", "", "Emotion preset (normal, happy, sad, angry, whisper, toneup, tonedown)")
	captionsCmd.Flags().Float64("emotion-intensity", -1, "Emotion intensity (0.0-2.0, default 1.0)")
	captionsCmd.Flags().String("prev-text", "", "Previous text for context")
	captionsCmd.Flags().String("next-text", "", "Next text for context")
	captionsCmd.Flags().Int("volume", -1, "Volume (0-200, default 100)")
	captionsCmd.Flags().Int("pitch", 0, "Pitch in semitones (-12 to +12, default 0)")
	captionsCmd.Flags().Float64("tempo", -1, "Tempo multiplier (0.5-2.0, default 1.0)")
	captionsCmd.Flags().String("audio-format", "", "Audio format (wav, mp3) — distinct from --format which is the caption format")
	captionsCmd.Flags().Int("seed", -1, "Random seed for reproducible output")

	rootCmd.AddCommand(captionsCmd)
}
