package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/hayasedb/hayase-cli/internal/extractors"
	"github.com/hayasedb/hayase-cli/internal/models"
	"github.com/hayasedb/hayase-cli/internal/players"
	"github.com/hayasedb/hayase-cli/internal/providers"
	"github.com/hayasedb/hayase-cli/internal/providers/aniworld"
	"github.com/hayasedb/hayase-cli/internal/storage"
	"github.com/hayasedb/hayase-cli/internal/tui/navigation"
	"github.com/hayasedb/hayase-cli/internal/tui/views"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
)

type PlaybackEndedMsg struct{}

type Model struct {
	state          *navigation.State
	provider       providers.Provider
	playerRegistry *players.Registry
	config         *storage.Config
	animeView      *views.AnimeView
	seasonView     *views.SeasonView
	episodeView    *views.EpisodeView
	playerView     *views.PlayerView
	ctx            context.Context
	cancelFunc     context.CancelFunc
	playbackCancel context.CancelFunc
}

func NewModel(
	ctx context.Context,
	cancelFunc context.CancelFunc,
	provider providers.Provider,
	playerRegistry *players.Registry,
	config *storage.Config,
) Model {
	state := navigation.NewState()

	model := Model{
		state:          state,
		provider:       provider,
		playerRegistry: playerRegistry,
		config:         config,
		animeView:      views.NewAnimeView(state, provider, config),
		seasonView:     views.NewSeasonView(state, provider),
		episodeView:    views.NewEpisodeView(state, provider),
		playerView:     views.NewPlayerView(state, provider),
		ctx:            ctx,
		cancelFunc:     cancelFunc,
	}

	return model
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			log.Debug("Ctrl+C pressed in TUI, cancelling context and exiting")
			if m.playbackCancel != nil {
				m.playbackCancel()
				m.playbackCancel = nil
			}
			if m.cancelFunc != nil {
				m.cancelFunc()
			}
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.state.SetDimensions(msg.Width, msg.Height)
		m.animeView, _ = m.animeView.Update(msg)
		m.seasonView, _ = m.seasonView.Update(msg)
		m.episodeView, _ = m.episodeView.Update(msg)
		return m, nil

	case views.PlayEpisodeMsg:
		return m, m.handlePlayback(msg.Anime, msg.Episode)

	case PlaybackEndedMsg:
		if m.state.GetCurrentView() == navigation.PlayerView {
			m.state.NavigateBack()
		}
		return m, nil
	}

	currentView := m.state.GetCurrentView()
	previousView := currentView

	var cmd tea.Cmd
	switch currentView {
	case navigation.AnimeView:
		m.animeView, cmd = m.animeView.Update(msg)
	case navigation.SeasonView:
		m.seasonView, cmd = m.seasonView.Update(msg)
	case navigation.EpisodeView:
		m.episodeView, cmd = m.episodeView.Update(msg)
	case navigation.PlayerView:
		m.playerView, cmd = m.playerView.Update(msg)
		if m.state.GetCurrentView() != navigation.PlayerView && m.playbackCancel != nil {
			m.playbackCancel()
			m.playbackCancel = nil
		}
	}

	if m.state.GetCurrentView() != previousView {
		cmd = tea.Batch(cmd, m.handleViewTransition(m.state.GetCurrentView()))
	}

	return m, cmd
}

func (m Model) handleViewTransition(to navigation.ViewState) tea.Cmd {
	switch to {
	case navigation.AnimeView:
	case navigation.SeasonView:
		if anime := m.state.GetAnime(); anime != nil {
			return m.seasonView.LoadAnime(anime)
		}
	case navigation.EpisodeView:
		m.episodeView.LoadEpisodes()
	case navigation.PlayerView:
		return m.playerView.Init()
	}
	return nil
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

func (m Model) View() string {
	if m.state.IsQuitting() {
		return "\n  Goodbye!\n\n"
	}

	switch m.state.GetCurrentView() {
	case navigation.AnimeView:
		return m.animeView.View()
	case navigation.SeasonView:
		return m.seasonView.View()
	case navigation.EpisodeView:
		return m.episodeView.View()
	case navigation.PlayerView:
		return m.playerView.View()
	default:
		return m.animeView.View()
	}
}

func (m *Model) handlePlayback(anime *models.Anime, episode *models.Episode) tea.Cmd {
	return func() tea.Msg {
		log.Info("Starting playback",
			"anime", anime.Title,
			"season", episode.Season,
			"episode", episode.Episode)

		if m.playbackCancel != nil {
			m.playbackCancel()
		}

		playbackCtx, cancel := context.WithCancel(m.ctx)
		m.playbackCancel = cancel
		ctx := playbackCtx
		maxRetries := 3

		episodeDetails, err := m.provider.GetEpisode(ctx, anime, episode.Season, episode.Episode)
		if err != nil {
			log.Error("Failed to get episode details", "error", err)
			return tea.Quit()
		}

		player, err := m.playerRegistry.GetDefault()
		if err != nil {
			log.Error("No player available", "error", err)
			return tea.Quit()
		}

		preferredLang := m.config.GetLanguage()

		availableProviders := filterAvailableProviders(episodeDetails.Providers)

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
			for name, languages := range availableProviders {
				for _, url := range languages {
					redirectURL = url
					providerName = name
					found = true
					break
				}
				if found {
					break
				}
			}
		}

		if !found {
			log.Error("No stream providers available")
			return tea.Quit()
		}

		aniWorldProvider, ok := m.provider.(*aniworld.Provider)
		if !ok {
			log.Error("Provider is not AniWorld provider")
			return tea.Quit()
		}

		title := fmt.Sprintf("%s - %s", anime.Title, episode.String())

		for attempt := 1; attempt <= maxRetries; attempt++ {
			log.Info("Extracting stream URL",
				"provider", providerName,
				"attempt", attempt)

			streamURL, err := aniWorldProvider.GetClient().ExtractStreamURL(ctx, redirectURL)
			if err != nil {
				log.Warn("Failed to extract stream URL", "error", err, "attempt", attempt)
				if attempt == maxRetries {
					log.Error("Failed to extract stream URL after all retries")
					return tea.Quit()
				}
				continue
			}

			if streamURL.IsExpired() {
				log.Warn("Stream URL is expired", "attempt", attempt)
				if attempt == maxRetries {
					log.Error("Stream URL still expired after all retries")
					return tea.Quit()
				}
				continue
			}

			log.Info("Starting playback",
				"quality", streamURL.Quality.String(),
				"player", player.Name())

			playbackErr := player.Play(ctx, streamURL, title)
			if playbackErr == nil {
				return PlaybackEndedMsg{}
			}

			errorStr := playbackErr.Error()
			isRetryable := strings.Contains(errorStr, "expired") ||
				strings.Contains(errorStr, "403") ||
				strings.Contains(errorStr, "Forbidden")

			if isRetryable && attempt < maxRetries {
				log.Warn("Playback failed with retryable error", "error", playbackErr, "attempt", attempt)
				continue
			}

			log.Error("Failed to play episode", "error", playbackErr)
			return PlaybackEndedMsg{}
		}

		return PlaybackEndedMsg{}
	}
}
