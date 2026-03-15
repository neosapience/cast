package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".typecast", "config.yaml"), nil
}

func readConfig() (map[string]any, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	config := map[string]any{}
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	return config, nil
}

func writeConfig(config map[string]any) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func saveConfig(apiKey string) error {
	config, err := readConfig()
	if err != nil {
		return err
	}
	config["api_key"] = apiKey
	return writeConfig(config)
}

// configKeys maps CLI flag names to config file keys
var configKeys = map[string]string{
	"voice-id":          "voice_id",
	"model":             "model",
	"language":          "language",
	"emotion":           "emotion",
	"emotion-preset":    "emotion_preset",
	"emotion-intensity": "emotion_intensity",
	"volume":            "volume",
	"pitch":             "pitch",
	"tempo":             "tempo",
	"format":            "format",
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage default settings",
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a default value",
	Long: "Set a default value in ~/.typecast/config.yaml\n\nAvailable keys: " +
		"voice-id, model, language, emotion, emotion-preset, emotion-intensity, volume, pitch, tempo, format",
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]

		configKey, ok := configKeys[key]
		if !ok {
			return fmt.Errorf("unknown key %q, available: %s", key, strings.Join(availableKeys(), ", "))
		}

		config, err := readConfig()
		if err != nil {
			return err
		}

		config[configKey] = value
		if err := writeConfig(config); err != nil {
			return err
		}

		fmt.Printf("%s = %s\n", configKey, value)
		return nil
	},
}

var configUnsetCmd = &cobra.Command{
	Use:   "unset <key>",
	Short: "Remove a default value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		configKey, ok := configKeys[key]
		if !ok {
			return fmt.Errorf("unknown key %q, available: %s", key, strings.Join(availableKeys(), ", "))
		}

		config, err := readConfig()
		if err != nil {
			return err
		}

		delete(config, configKey)
		if err := writeConfig(config); err != nil {
			return err
		}

		fmt.Printf("unset %s\n", configKey)
		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show current config",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := readConfig()
		if err != nil {
			return err
		}

		if len(config) == 0 {
			fmt.Println("(empty)")
			return nil
		}

		keys := make([]string, 0, len(config))
		for k := range config {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			v := config[k]
			if k == "api_key" {
				s := fmt.Sprintf("%v", v)
				if len(s) > 8 {
					v = s[:8] + "..."
				}
			}
			fmt.Printf("%s = %v\n", k, v)
		}
		return nil
	},
}

func availableKeys() []string {
	keys := make([]string, 0, len(configKeys))
	for k := range configKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configUnsetCmd)
	configCmd.AddCommand(configListCmd)
	rootCmd.AddCommand(configCmd)
}
