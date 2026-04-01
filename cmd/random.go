package cmd

import (
	"fmt"
	"math/rand"

	"github.com/neosapience/cast/internal/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var randomCmd = &cobra.Command{
	Use:   "random",
	Short: "Pick a random voice from the filtered pool",
	RunE: func(cmd *cobra.Command, args []string) error {
		flags := cmd.Flags()
		model, _ := flags.GetString("model")
		gender, _ := flags.GetString("gender")
		age, _ := flags.GetString("age")
		useCase, _ := flags.GetString("use-case")
		name, _ := flags.GetString("name")
		emotion, _ := flags.GetString("emotion")

		baseURL := viper.GetString("base_url")
		var c *client.Client
		if baseURL != "" {
			c = client.NewWithBaseURL(viper.GetString("api_key"), baseURL)
		} else {
			c = client.New(viper.GetString("api_key"))
		}

		voices, err := c.ListVoices(client.ListVoicesParams{
			Model:   model,
			Gender:  gender,
			Age:     age,
			UseCase: useCase,
		})
		if err != nil {
			return err
		}

		if name != "" {
			voices = filterVoicesByName(voices, name)
		}
		if emotion != "" {
			voices = filterVoicesByEmotion(voices, emotion)
		}

		if len(voices) == 0 {
			return fmt.Errorf("no voices found matching the given filters")
		}

		v := voices[rand.Intn(len(voices))]
		fmt.Println(v.VoiceID)
		return nil
	},
}

func init() {
	randomCmd.Flags().String("model", "", "Filter by model (ssfm-v30, ssfm-v21)")
	randomCmd.Flags().String("gender", "", "Filter by gender (male, female)")
	randomCmd.Flags().String("age", "", "Filter by age (child, teenager, young_adult, middle_age, elder)")
	randomCmd.Flags().String("use-case", "", "Filter by use case")
	randomCmd.Flags().String("name", "", "Filter by name (case-insensitive substring match)")
	randomCmd.Flags().String("emotion", "", "Filter by supported emotion")
}
