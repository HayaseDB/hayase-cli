package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hayasedb/hayase-cli/internal/storage"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View or manage configuration",
	Long: `View current configuration or set configuration values.

Examples:
  hayase-cli config                           # Show current config
  hayase-cli config set language eng-sub     # Set preferred language
  hayase-cli config set quality 720p         # Set preferred quality
  hayase-cli config set provider aniworld    # Set preferred provider`,

	RunE: runConfig,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value.

Available settings:
  language    Preferred language (ger-sub, eng-sub, ger-dub)
  quality     Preferred quality (720p, 1080p, 1440p, 2160p)  
  provider    Preferred provider (aniworld)
  player      Preferred player (mpv)
  timeout     Request timeout in seconds`,

	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd)
}

func runConfig(*cobra.Command, []string) error {
	config, err := storage.NewConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println("current config:")
	fmt.Println()

	fmt.Printf("  Provider:  %s\n", config.GetProvider())
	fmt.Printf("  Language:  %s\n", config.GetLanguage().String())
	fmt.Printf("  Quality:   %s\n", config.GetQuality().String())
	fmt.Printf("  Player:    %s\n", config.GetPlayer())
	fmt.Printf("  Timeout:   %d seconds\n", config.GetTimeout())

	return nil
}

func runConfigSet(_ *cobra.Command, args []string) error {
	key := strings.ToLower(args[0])
	value := strings.ToLower(args[1])

	config, err := storage.NewConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	switch key {
	case "language":
		validLanguages := []string{"ger-sub", "eng-sub", "ger-dub"}
		if !contains(validLanguages, value) {
			return fmt.Errorf("invalid language '%s'. Valid options: %s", value, strings.Join(validLanguages, ", "))
		}
		config.Set("language", value)

	case "quality":
		validQualities := []string{"720p", "1080p", "1440p", "2160p"}
		if !contains(validQualities, value) {
			return fmt.Errorf("invalid quality '%s'. Valid options: %s", value, strings.Join(validQualities, ", "))
		}
		config.Set("quality", value)

	case "provider":
		validProviders := []string{"aniworld"}
		if !contains(validProviders, value) {
			return fmt.Errorf("invalid provider '%s'. Valid options: %s", value, strings.Join(validProviders, ", "))
		}
		config.Set("provider", value)

	case "player":
		validPlayers := []string{"mpv"}
		if !contains(validPlayers, value) {
			return fmt.Errorf("invalid player '%s'. Valid options: %s", value, strings.Join(validPlayers, ", "))
		}
		config.Set("player", value)

	case "timeout":
		config.Set("timeout", value)

	default:
		return fmt.Errorf("unknown configuration key '%s'", key)
	}

	if err := config.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("Configuration updated: %s = %s\n", key, value)

	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
