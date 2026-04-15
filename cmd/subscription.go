package cmd

import (
	"fmt"

	"github.com/neosapience/cast/internal/client"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var subscriptionCmd = &cobra.Command{
	Use:   "subscription",
	Short: "Show current subscription details",
	RunE: func(cmd *cobra.Command, args []string) error {
		baseURL := viper.GetString("base_url")
		var c *client.Client
		if baseURL != "" {
			c = client.NewWithBaseURL(viper.GetString("api_key"), baseURL)
		} else {
			c = client.New(viper.GetString("api_key"))
		}

		sub, err := c.GetMySubscription()
		if err != nil {
			return err
		}

		remaining := sub.Credits.PlanCredits - sub.Credits.UsedCredits

		fmt.Printf("Plan:        %s\n", sub.Plan)
		fmt.Printf("Credits:     %s / %s used\n",
			formatInt(sub.Credits.UsedCredits),
			formatInt(sub.Credits.PlanCredits))
		fmt.Printf("Remaining:   %s\n", formatInt(remaining))
		fmt.Printf("Concurrency: %d\n", sub.Limits.ConcurrencyLimit)
		return nil
	},
}

func formatInt(n int64) string {
	if n < 0 {
		return "-" + formatInt(-n)
	}
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

func init() {
	rootCmd.AddCommand(subscriptionCmd)
}
