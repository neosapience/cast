package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/neosapience/cast/internal/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var voicesCloneCmd = &cobra.Command{
	Use:   "clone <audio_file>",
	Short: "Clone a voice from a WAV or MP3 sample",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		model, _ := cmd.Flags().GetString("model")
		professional, _ := cmd.Flags().GetBool("professional")
		language, _ := cmd.Flags().GetString("language")
		asJSON, _ := cmd.Flags().GetBool("json")

		c := newTypecastClient()
		if professional {
			voice, err := c.CreateProfessionalVoice(client.ProfessionalCloneRequest{
				Name: name, Language: language, Model: model, AudioFilePath: args[0],
			})
			if err != nil {
				return err
			}
			if asJSON {
				return json.NewEncoder(os.Stdout).Encode(voice)
			}
			fmt.Println(voice.VoiceID)
			return nil
		}
		voice, err := c.CloneVoice(client.CloneVoiceRequest{
			Name:          name,
			Model:         model,
			AudioFilePath: args[0],
		})
		if err != nil {
			return err
		}

		if asJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(voice)
		}

		fmt.Println(voice.VoiceID)
		return nil
	},
}

var voicesCloneStatusCmd = &cobra.Command{
	Use:   "status <voice_id>",
	Short: "Get professional clone status",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		voice, err := newTypecastClient().GetCustomVoice(args[0])
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(voice)
	},
}

var voicesDeleteCmd = &cobra.Command{
	Use:   "delete <voice_id>",
	Short: "Delete a quick-cloned voice",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		voiceID := args[0]
		if !strings.HasPrefix(voiceID, "uc_") {
			return fmt.Errorf("only cloned voice IDs that start with 'uc_' can be deleted")
		}

		c := newTypecastClient()
		if err := c.DeleteClonedVoice(voiceID); err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "deleted %s\n", voiceID)
		return nil
	},
}

func newTypecastClient() *client.Client {
	baseURL := viper.GetString("base_url")
	if baseURL != "" {
		return client.NewWithBaseURL(viper.GetString("api_key"), baseURL)
	}
	return client.New(viper.GetString("api_key"))
}

func init() {
	voicesCloneCmd.Flags().String("name", "", "Display name for the cloned voice (1-30 characters)")
	_ = voicesCloneCmd.MarkFlagRequired("name")
	voicesCloneCmd.Flags().String("model", defaultModel, "Voice cloning model (ssfm-v30)")
	voicesCloneCmd.Flags().Bool("professional", false, "Create an asynchronous professional clone")
	voicesCloneCmd.Flags().String("language", "", "Language code required for professional cloning (for example: eng, kor)")
	voicesCloneCmd.Flags().Bool("json", false, "Output clone response as JSON")

	voicesCloneCmd.AddCommand(voicesCloneStatusCmd)
	voicesCmd.AddCommand(voicesCloneCmd)
	voicesCmd.AddCommand(voicesDeleteCmd)
}
