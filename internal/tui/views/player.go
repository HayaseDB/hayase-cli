package views

import (
	"fmt"

	"github.com/hayasedb/hayase-cli/internal/models"
	"github.com/hayasedb/hayase-cli/internal/providers"
	"github.com/hayasedb/hayase-cli/internal/tui/navigation"

	tea "github.com/charmbracelet/bubbletea"
)

type PlayEpisodeMsg struct {
	Anime    *models.Anime
	Episode  *models.Episode
	Provider providers.Provider
}

type PlayerView struct {
	state    *navigation.State
	provider providers.Provider
}

func NewPlayerView(state *navigation.State, provider providers.Provider) *PlayerView {
	return &PlayerView{
		state:    state,
		provider: provider,
	}
}

func (v *PlayerView) Init() tea.Cmd {
	anime := v.state.GetAnime()
	episode := v.state.GetEpisode()
	if anime != nil && episode != nil {
		return v.triggerPlayback(anime, episode)
	}
	return nil
}

func (v *PlayerView) triggerPlayback(anime *models.Anime, episode *models.Episode) tea.Cmd {
	return func() tea.Msg {
		return PlayEpisodeMsg{
			Anime:    anime,
			Episode:  episode,
			Provider: v.provider,
		}
	}
}

func (v *PlayerView) Update(msg tea.Msg) (*PlayerView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return v.handleKeys(msg)
	}

	return v, nil
}

func (v *PlayerView) handleKeys(msg tea.KeyMsg) (*PlayerView, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		v.state.SetQuitting(true)
		return v, tea.Quit
	case "esc":
		v.state.NavigateBack()
		return v, nil
	}
	return v, nil
}

func (v *PlayerView) View() string {
	anime := v.state.GetAnime()
	season := v.state.GetSeason()
	episode := v.state.GetEpisode()

	if anime == nil || episode == nil {
		return "Error: Missing selection data\n\nPress Esc to go back"
	}

	seasonText := fmt.Sprintf("Season %d", season)
	if season == 0 {
		seasonText = "Movie"
	}

	episodeInfo := fmt.Sprintf("Episode %d", episode.Episode)
	if episode.Title != "" {
		episodeInfo = fmt.Sprintf("Episode %d: %s", episode.Episode, episode.Title)
	}

	return fmt.Sprintf("Now Playing: %s - %s %s\nStatus: MPV Running\n\nPress Esc to go back, q to quit",
		anime.Title, seasonText, episodeInfo)
}
