package models

type Anime struct {
	ID       string
	Name     string
	Desc     string
	Year     int
	Seasons  int
	Episodes []Episode
	Genres   []string
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
