package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login [api_key]",
	Short: "Save API key to ~/.typecast/config.yaml",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var apiKey string

		if len(args) == 1 {
			apiKey = args[0]
		} else {
			const apiKeyURL = "https://typecast.ai/developers/api/api-key?utm_source=cast&utm_medium=cli"
			if err := openBrowser(apiKeyURL); err != nil {
				fmt.Printf("Open this URL to get your API key:\n  %s\n", apiKeyURL)
			} else {
				fmt.Println("Opening browser to get your API key...")
			}
			fmt.Print("Enter API key: ")
			fmt.Scanln(&apiKey)
		}

		if apiKey == "" {
			return fmt.Errorf("API key cannot be empty")
		}

		if err := saveConfig(apiKey); err != nil {
			return err
		}

		path, _ := configPath()
		fmt.Fprintf(os.Stderr, "API key saved to %s\n", path)
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove saved API key",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := readConfig()
		if err != nil {
			return err
		}

		if _, ok := config["api_key"]; !ok {
			fmt.Println("not logged in")
			return nil
		}

		delete(config, "api_key")
		if err := writeConfig(config); err != nil {
			return err
		}

		fmt.Println("logged out")
		return nil
	},
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
			return fmt.Errorf("no display server")
		}
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
}

func init() {
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
}
