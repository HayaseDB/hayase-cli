package tui

type Anime struct {
	ID       string
	Name     string
	Desc     string
	Year     int
	Seasons  int
	Episodes []Episode
	Genres   []string
	Rating   float64
}

type Episode struct {
	Season   int
	Number   int
	Title    string
	Duration string
}

type WatchProgress struct {
	Anime       *Anime
	LastEpisode int
	LastSeason  int
	ProgressPct int
}

func (a Anime) FilterValue() string {
	return a.Name
}

func (a Anime) Title() string {
	return a.Name
}

func (a Anime) Description() string {
	return a.Desc
}
