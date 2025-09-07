package models

import (
	"fmt"
	"time"
)

type Anime struct {
	Title       string    `json:"title"`
	Slug        string    `json:"slug"`
	Link        string    `json:"link"`
	Description string    `json:"description"`
	Year        int       `json:"year"`
	Episodes    []Episode `json:"episodes"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (a *Anime) String() string {
	if a.Year > 0 {
		return fmt.Sprintf("%s (%d)", a.Title, a.Year)
	}
	return a.Title
}

func (a *Anime) GetAvailableSeasons() []int {
	seasonSet := make(map[int]bool)
	for _, ep := range a.Episodes {
		seasonSet[ep.Season] = true
	}

	var seasons []int
	for season := range seasonSet {
		seasons = append(seasons, season)
	}
	return seasons
}

func (a *Anime) GetEpisodesForSeason(season int) []Episode {
	var episodes []Episode
	for _, ep := range a.Episodes {
		if ep.Season == season {
			episodes = append(episodes, ep)
		}
	}
	return episodes
}

type Episode struct {
	Season    int                            `json:"season"`
	Episode   int                            `json:"episode"`
	Title     string                         `json:"title"`
	Link      string                         `json:"link"`
	Providers map[string]map[Language]string `json:"providers"`
	Anime     *Anime                         `json:"-"`
	UpdatedAt time.Time                      `json:"updated_at"`
}

func (e *Episode) String() string {
	if e.Title != "" {
		return fmt.Sprintf("S%02dE%02d: %s", e.Season, e.Episode, e.Title)
	}
	return fmt.Sprintf("S%02dE%02d", e.Season, e.Episode)
}

type SearchResult struct {
	Anime *Anime  `json:"anime"`
	Score float64 `json:"score"`
}
