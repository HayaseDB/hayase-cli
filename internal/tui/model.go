package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type View int

const (
	BrowseView View = iota
	DetailsView
)

const visibleCards = 4

type Section int

const (
	SearchSection Section = iota
	ContinueSection
	TrendingSection
	NewReleasesSection
	SearchResultsSection
)

type Model struct {
	view View

	keys KeyMap

	activeSection     Section
	sectionCursors    map[Section]int
	sectionPaginators map[Section]paginator.Model
	visibleCards      int

	searchInput   textinput.Model
	searchActive  bool
	searchResults []Anime

	anime          []Anime
	continueList   []WatchProgress
	trendingList   []Anime
	newReleaseList []Anime

	selectedAnime   *Anime
	activeSeason    int
	seasonTabOffset int
	episodeCursor   int
	episodeOffset   int
	visibleEpisodes int

	width  int
	height int

	quitting bool

	titleScrollOffset int
}

type titleTickMsg time.Time

func titleTick() tea.Cmd {
	return tea.Tick(time.Millisecond*300, func(t time.Time) tea.Msg {
		return titleTickMsg(t)
	})
}

func newPaginator(totalItems int) paginator.Model {
	p := paginator.New()
	p.Type = paginator.Dots
	p.PerPage = visibleCards
	p.SetTotalPages(totalItems)
	p.ActiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "252"}).Render("•")
	p.InactiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Render("•")
	return p
}

func New() Model {
	var anime []Anime
	var continueList []WatchProgress
	var trendingList []Anime
	var newReleaseList []Anime

	ti := textinput.New()
	ti.Placeholder = "Search anime..."
	ti.CharLimit = 50
	ti.Width = 30
	ti.PromptStyle = lipgloss.NewStyle().Foreground(TextMuted)
	ti.TextStyle = lipgloss.NewStyle().Foreground(Text)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(TextDim)

	paginators := make(map[Section]paginator.Model)
	paginators[ContinueSection] = newPaginator(len(continueList))
	paginators[TrendingSection] = newPaginator(len(trendingList))
	paginators[NewReleasesSection] = newPaginator(len(newReleaseList))
	paginators[SearchResultsSection] = newPaginator(len(anime))

	return Model{
		view:              BrowseView,
		keys:              DefaultKeyMap(),
		activeSection:     SearchSection,
		sectionCursors:    make(map[Section]int),
		sectionPaginators: paginators,
		visibleCards:      visibleCards,

		searchInput: ti,

		anime:          anime,
		continueList:   continueList,
		trendingList:   trendingList,
		newReleaseList: newReleaseList,

		visibleEpisodes: 10,
	}
}

func (m Model) Init() tea.Cmd {
	return titleTick()
}

func (m Model) getSectionLength(section Section) int {
	switch section {
	case ContinueSection:
		return len(m.continueList)
	case TrendingSection:
		return len(m.trendingList)
	case NewReleasesSection:
		return len(m.newReleaseList)
	case SearchResultsSection:
		return len(m.searchResults)
	default:
		return 0
	}
}

func (m Model) getSelectedAnime() *Anime {
	switch m.activeSection {
	case ContinueSection:
		cursor := m.sectionCursors[ContinueSection]
		if cursor < len(m.continueList) {
			return m.continueList[cursor].Anime
		}
	case TrendingSection:
		cursor := m.sectionCursors[TrendingSection]
		if cursor < len(m.trendingList) {
			return &m.trendingList[cursor]
		}
	case NewReleasesSection:
		cursor := m.sectionCursors[NewReleasesSection]
		if cursor < len(m.newReleaseList) {
			return &m.newReleaseList[cursor]
		}
	case SearchResultsSection:
		cursor := m.sectionCursors[SearchResultsSection]
		if cursor < len(m.searchResults) {
			return &m.searchResults[cursor]
		}
	}
	return nil
}

type CardData struct {
	Name     string
	Subtitle string
	Info     string
}

func (m Model) getCardData(section Section, index int) CardData {
	switch section {
	case ContinueSection:
		if index < len(m.continueList) {
			wp := m.continueList[index]
			return CardData{
				Name:     wp.Anime.Name,
				Subtitle: fmt.Sprintf("S%d E%d", wp.LastSeason, wp.LastEpisode),
				Info:     ProgressBar(wp.ProgressPct, CardWidth-2),
			}
		}
	case TrendingSection:
		if index < len(m.trendingList) {
			anime := m.trendingList[index]
			return CardData{
				Name:     anime.Name,
				Subtitle: fmt.Sprintf("%d", anime.Year),
				Info:     CardInfoStyle.Render(fmt.Sprintf("%d Seasons", anime.Seasons)),
			}
		}
	case NewReleasesSection:
		if index < len(m.newReleaseList) {
			anime := m.newReleaseList[index]
			info := ""
			if len(anime.Genres) > 0 {
				info = CardInfoStyle.Render(anime.Genres[0])
			}
			return CardData{
				Name:     anime.Name,
				Subtitle: fmt.Sprintf("%d", anime.Year),
				Info:     info,
			}
		}
	case SearchResultsSection:
		if index < len(m.searchResults) {
			anime := m.searchResults[index]
			return CardData{
				Name:     anime.Name,
				Subtitle: fmt.Sprintf("%d", anime.Year),
				Info:     CardInfoStyle.Render(fmt.Sprintf("%d Seasons", anime.Seasons)),
			}
		}
	}
	return CardData{}
}

func filterAnime(anime []Anime, query string) []Anime {
	if query == "" {
		return anime
	}
	query = strings.ToLower(query)
	var filtered []Anime
	for _, a := range anime {
		if strings.Contains(strings.ToLower(a.Name), query) {
			filtered = append(filtered, a)
		}
	}
	return filtered
}

func (m Model) calculateVisibleCards() int {
	availableWidth := m.width - 6
	cardTotalWidth := CardWidth + 2
	visible := availableWidth / cardTotalWidth
	if visible < 1 {
		visible = 1
	}
	return visible
}

func (m Model) getSeasonEpisodes() []Episode {
	if m.selectedAnime == nil {
		return nil
	}
	season := m.activeSeason + 1
	var episodes []Episode
	for _, ep := range m.selectedAnime.Episodes {
		if ep.Season == season {
			episodes = append(episodes, ep)
		}
	}
	return episodes
}
