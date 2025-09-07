package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"

	"github.com/hayasedb/hayase-cli/internal/extractors"
	"github.com/hayasedb/hayase-cli/internal/models"
	"github.com/hayasedb/hayase-cli/internal/players"
	"github.com/hayasedb/hayase-cli/internal/players/mpv"
	"github.com/hayasedb/hayase-cli/internal/providers"
	"github.com/hayasedb/hayase-cli/internal/providers/aniworld"
	"github.com/hayasedb/hayase-cli/internal/storage"
	"github.com/hayasedb/hayase-cli/internal/tui/app"
)

var (
	animeName  string
	seasonNum  int
	episodeNum int
	debug      bool
)

var rootCmd = &cobra.Command{
	Use:     "hayase-cli",
	Short:   "Stream anime from your terminal",
	Version: Version,
	Long: `hayase-cli A modern, fast anime streaming application for the terminal.

Search for anime, select episodes, and stream directly using your preferred player.
Supports progressive disclosure: use flags to skip TUI screens.

Examples:
  hayase-cli                                    # Full TUI experience
  hayase-cli --anime "Attack on Titan"         # Skip search
  hayase-cli --anime "Naruto" --season 1       # Skip search and season
  hayase-cli --anime "One Piece" -s 1 -e 1     # Direct play`,

	RunE: runWatch,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().StringVarP(&animeName, "anime", "a", "", "Anime name to search/play")
	rootCmd.Flags().IntVarP(&seasonNum, "season", "s", 0, "Season number")
	rootCmd.Flags().IntVarP(&episodeNum, "episode", "e", 0, "Episode number")
	rootCmd.Flags().BoolVar(&debug, "debug", false, "Enable debug logging")
}

func runWatch(*cobra.Command, []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	needsTUI := animeName == "" || (animeName != "" && seasonNum == 0) || (animeName != "" && seasonNum > 0 && episodeNum == 0)

	if needsTUI && !canAccessTTY() {
		fmt.Println("Warning: TUI mode requires a proper terminal environment.")
		fmt.Println("   Use direct playback instead:")
		fmt.Println("   ./hayase-cli --anime \"Anime Name\" --season 1 --episode 1")
		fmt.Println("   Or use --help for more options.")
		return fmt.Errorf("TTY not available for interactive mode")
	}

	logFile, err := setupFileLogging()
	if err != nil {
		fmt.Printf("Warning: Could not setup file logging: %v\n", err)
	} else {
		defer func() {
			if err := logFile.Close(); err != nil {
				log.Debug("Failed to close log file", "error", err)
			}
		}()
		log.Info("Starting hayase-cli", "version", Version, "timestamp", time.Now())
	}

	if debug {
		log.SetLevel(log.DebugLevel)
		log.Debug("Debug logging enabled")
	} else {
		log.SetLevel(log.InfoLevel)
	}

	config, err := storage.NewConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	providerRegistry := providers.NewRegistry()
	providerRegistry.Register("aniworld", aniworld.New())

	provider, err := providerRegistry.GetDefault()
	if err != nil {
		return fmt.Errorf("no provider available: %w", err)
	}

	playerRegistry := players.NewRegistry()
	playerRegistry.Register("mpv", mpv.New())

	if _, err := playerRegistry.GetDefault(); err != nil {
		return fmt.Errorf("no player available: %w", err)
	}

	if animeName != "" && seasonNum > 0 && episodeNum > 0 {
		return playDirect(ctx, provider, playerRegistry, config, animeName, seasonNum, episodeNum)
	}

	model := app.NewModel(ctx, cancel, provider, playerRegistry, config)

	p := tea.NewProgram(
		&model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	_, err = p.Run()
	return err
}

func filterAvailableProviders(providers map[string]map[models.Language]string) map[string]map[models.Language]string {
	extractorSystem := extractors.NewSystem(nil)
	filtered := make(map[string]map[models.Language]string)

	supportedNames := make(map[string]bool)
	for _, extractor := range extractorSystem.GetExtractors() {
		supportedNames[extractor.Name()] = true
	}

	for providerName, languages := range providers {
		if supportedNames[providerName] {
			filtered[providerName] = languages
		}
	}

	return filtered
}

func playDirect(ctx context.Context, provider providers.Provider, playerRegistry *players.Registry, config *storage.Config, animeName string, seasonNum, episodeNum int) error {
	fmt.Printf("Searching for: %s\n", animeName)

	results, err := provider.Search(ctx, animeName)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		return fmt.Errorf("no anime found for: %s", animeName)
	}

	anime := results[0].Anime
	fmt.Printf("Found: %s\n", anime.String())

	if err := provider.GetEpisodes(ctx, anime); err != nil {
		return fmt.Errorf("failed to get episodes: %w", err)
	}

	episode, err := provider.GetEpisode(ctx, anime, seasonNum, episodeNum)
	if err != nil {
		return fmt.Errorf("episode S%02dE%02d not found: %w", seasonNum, episodeNum, err)
	}

	fmt.Printf("Playing: %s\n", episode.String())

	preferredLang := config.GetLanguage()

	availableProviders := filterAvailableProviders(episode.Providers)

	var redirectURL string
	var providerName string
	var found bool

	for name, languages := range availableProviders {
		if url, exists := languages[preferredLang]; exists {
			redirectURL = url
			providerName = name
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("no stream available for this episode")
	}

	fmt.Printf("Extracting stream from %s...\n", providerName)

	aniWorldProvider, ok := provider.(*aniworld.Provider)
	if !ok {
		return fmt.Errorf("provider is not AniWorld provider")
	}

	streamURL, err := aniWorldProvider.GetClient().ExtractStreamURL(ctx, redirectURL)
	if err != nil {
		return fmt.Errorf("failed to extract stream URL: %w", err)
	}

	fmt.Printf("Stream extracted: %s quality\n", streamURL.Quality.String())

	player, err := playerRegistry.GetDefault()
	if err != nil {
		return fmt.Errorf("no player available: %w", err)
	}

	title := fmt.Sprintf("%s - %s", anime.Title, episode.String())
	fmt.Printf("Starting playback with %s...\n", player.Name())

	err = player.Play(ctx, streamURL, title)
	if err != nil {
		return fmt.Errorf("playback failed: %w", err)
	}

	fmt.Println("Playback completed!")

	return nil
}

func canAccessTTY() bool {
	if file, err := os.OpenFile("/dev/tty", os.O_RDWR, 0); err == nil {
		func() {
			if err := file.Close(); err != nil {
				log.Debug("Failed to close TTY file", "error", err)
			}
		}()
		return true
	}

	if fi, err := os.Stdin.Stat(); err == nil {
		if (fi.Mode() & os.ModeCharDevice) != 0 {
			return true
		}
	}

	return false
}

func setupFileLogging() (*os.File, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not get home directory: %w", err)
	}

	logDir := filepath.Join(homeDir, ".config", "hayase-cli", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("could not create log directory: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logPath := filepath.Join(logDir, fmt.Sprintf("hayase-%s.log", timestamp))

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("could not open log file: %w", err)
	}

	log.SetOutput(logFile)

	fmt.Printf("Logging to: %s\n", logPath)

	return logFile, nil
}
