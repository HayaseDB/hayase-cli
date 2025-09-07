package navigation

import (
	"github.com/hayasedb/hayase-cli/internal/models"
)

type ViewState int

const (
	AnimeView ViewState = iota
	SeasonView
	EpisodeView
	PlayerView
)

type State struct {
	current   ViewState
	selection struct {
		Anime   *models.Anime
		Season  int
		Episode *models.Episode
	}
	width    int
	height   int
	quitting bool
}

func NewState() *State {
	return &State{
		current: AnimeView,
	}
}

func (s *State) GetCurrentView() ViewState {
	return s.current
}

func (s *State) SetCurrentView(view ViewState) {
	s.current = view
}

func (s *State) GetAnime() *models.Anime {
	return s.selection.Anime
}

func (s *State) SetAnime(anime *models.Anime) {
	s.selection.Anime = anime
}

func (s *State) GetSeason() int {
	return s.selection.Season
}

func (s *State) SetSeason(season int) {
	s.selection.Season = season
}

func (s *State) GetEpisode() *models.Episode {
	return s.selection.Episode
}

func (s *State) SetEpisode(episode *models.Episode) {
	s.selection.Episode = episode
}

func (s *State) ClearAnime() {
	s.selection.Anime = nil
}

func (s *State) ClearSeason() {
	s.selection.Season = 0
}

func (s *State) ClearEpisode() {
	s.selection.Episode = nil
}

func (s *State) SetDimensions(width, height int) {
	s.width = width
	s.height = height
}

func (s *State) GetDimensions() (int, int) {
	return s.width, s.height
}

func (s *State) SetQuitting(quit bool) {
	s.quitting = quit
}

func (s *State) IsQuitting() bool {
	return s.quitting
}

func (s *State) NavigateForward() {
	switch s.current {
	case AnimeView:
		s.current = SeasonView
	case SeasonView:
		s.current = EpisodeView
	case EpisodeView:
		s.current = PlayerView
	case PlayerView:
	}
}

func (s *State) NavigateBack() {
	switch s.current {
	case AnimeView:
	case SeasonView:
		s.current = AnimeView
	case EpisodeView:
		s.current = SeasonView
	case PlayerView:
		s.current = EpisodeView
	}
}
