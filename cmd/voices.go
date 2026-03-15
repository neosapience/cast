package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/neosapience/cast/internal/client"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var voicesCmd = &cobra.Command{
	Use:   "voices",
	Short: "Manage voices",
}

var voicesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available voices",
	RunE: func(cmd *cobra.Command, args []string) error {
		flags := cmd.Flags()
		model, _ := flags.GetString("model")
		gender, _ := flags.GetString("gender")
		age, _ := flags.GetString("age")
		useCase, _ := flags.GetString("use-case")
		name, _ := flags.GetString("name")
		emotion, _ := flags.GetString("emotion")
		asJSON, _ := flags.GetBool("json")

		c := client.New(viper.GetString("api_key"))
		voices, err := c.ListVoices(client.ListVoicesParams{
			Model:   model,
			Gender:  gender,
			Age:     age,
			UseCase: useCase,
		})
		if err != nil {
			return err
		}

		// Client-side filters (not supported by API).
		if name != "" {
			voices = filterVoicesByName(voices, name)
		}
		if emotion != "" {
			voices = filterVoicesByEmotion(voices, emotion)
		}

		if asJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(voices)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tGENDER\tAGE\tMODELS")
		for _, v := range voices {
			modelNames := make([]string, len(v.Models))
			for i, m := range v.Models {
				modelNames[i] = m.Version
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				v.VoiceID, v.VoiceName, v.Gender, v.Age,
				strings.Join(modelNames, ", "))
		}
		return w.Flush()
	},
}

func filterVoicesByName(voices []client.Voice, name string) []client.Voice {
	lower := strings.ToLower(name)
	out := voices[:0]
	for _, v := range voices {
		if strings.Contains(strings.ToLower(v.VoiceName), lower) {
			out = append(out, v)
		}
	}
	return out
}

func filterVoicesByEmotion(voices []client.Voice, emotion string) []client.Voice {
	out := voices[:0]
	for _, v := range voices {
		for _, m := range v.Models {
			if containsString(m.Emotions, emotion) {
				out = append(out, v)
				break
			}
		}
	}
	return out
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

var voicesGetCmd = &cobra.Command{
	Use:   "get <voice_id>",
	Short: "Get voice details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		asJSON, _ := cmd.Flags().GetBool("json")

		c := client.New(viper.GetString("api_key"))
		voice, err := c.GetVoice(args[0])
		if err != nil {
			return err
		}

		if asJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(voice)
		}

		fmt.Printf("ID:        %s\n", voice.VoiceID)
		fmt.Printf("Name:      %s\n", voice.VoiceName)
		fmt.Printf("Gender:    %s\n", voice.Gender)
		fmt.Printf("Age:       %s\n", voice.Age)
		fmt.Printf("Use Cases: %s\n", strings.Join(voice.UseCases, ", "))
		fmt.Println()
		fmt.Println("Models:")
		for _, m := range voice.Models {
			fmt.Printf("  %-10s  %s\n", m.Version, strings.Join(m.Emotions, ", "))
		}
		return nil
	},
}

func init() {
	voicesListCmd.Flags().String("model", "", "Filter by model (ssfm-v30, ssfm-v21)")
	voicesListCmd.Flags().String("gender", "", "Filter by gender (male, female)")
	voicesListCmd.Flags().String("age", "", "Filter by age (child, teenager, young_adult, middle_age, elder)")
	voicesListCmd.Flags().String("use-case", "", "Filter by use case (Announcer, Anime, Audiobook, Conversational, Documentary, E-learning, Rapper, Game, Tiktok/Reels, News, Podcast, Voicemail, Ads)")
	voicesListCmd.Flags().String("name", "", "Filter by name (case-insensitive substring match)")
	voicesListCmd.Flags().String("emotion", "", "Filter by supported emotion (normal, happy, sad, angry, whisper, toneup, tonedown)")
	voicesListCmd.Flags().Bool("json", false, "Output as JSON instead of table")

	voicesGetCmd.Flags().Bool("json", false, "Output as JSON instead of human-readable format")

	voicesCmd.AddCommand(voicesListCmd)
	voicesCmd.AddCommand(voicesGetCmd)
	rootCmd.AddCommand(voicesCmd)
}
