package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/hayasedb/hayase/internal/history"
	"github.com/hayasedb/hayase/internal/models"
	"github.com/hayasedb/hayase/internal/scraper"
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
	searchLoading bool

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

	scraper        scraper.Interface
	history        *history.Manager
	loading        bool
	loadingStatus  string
	loadError      error
	loadingDetails bool
	spinner        spinner.Model
}

type dataLoadedMsg struct {
	trending   []models.Anime
	newRelease []models.Anime
	err        error
}

type searchResultsMsg struct {
	results []models.Anime
	err     error
}

type animeDetailsMsg struct {
	anime *models.Anime
	err   error
}

type animeDetailUpdateMsg struct {
	slug  string
	anime *models.Anime
	err   error
}

type titleTickMsg time.Time

func titleTick() tea.Cmd {
	return tea.Tick(time.Millisecond*300, func(t time.Time) tea.Msg {
		return titleTickMsg(t)
	})
}

type searchDebounceMsg struct {
	query string
}

func searchDebounce(query string) tea.Cmd {
	return tea.Tick(time.Millisecond*200, func(_ time.Time) tea.Msg {
		return searchDebounceMsg{query: query}
	})
}

func newPaginator() paginator.Model {
	p := paginator.New()
	p.Type = paginator.Dots
	p.PerPage = visibleCards
	p.ActiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "252"}).Render("•")
	p.InactiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Render("•")
	return p
}

func New(s scraper.Interface, h *history.Manager) Model {
	ti := textinput.New()
	ti.Placeholder = "Search anime..."
	ti.CharLimit = 50
	ti.Width = 30
	ti.PromptStyle = lipgloss.NewStyle().Foreground(TextMuted)
	ti.TextStyle = lipgloss.NewStyle().Foreground(Text)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(TextDim)

	paginators := make(map[Section]paginator.Model)
	paginators[ContinueSection] = newPaginator()
	paginators[TrendingSection] = newPaginator()
	paginators[NewReleasesSection] = newPaginator()
	paginators[SearchResultsSection] = newPaginator()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(Primary)

	return Model{
		view:              BrowseView,
		keys:              DefaultKeyMap(),
		activeSection:     SearchSection,
		sectionCursors:    make(map[Section]int),
		sectionPaginators: paginators,
		visibleCards:      visibleCards,
		searchInput:       ti,
		visibleEpisodes:   10,
		scraper:           s,
		history:           h,
		loading:           true,
		loadingStatus:     "Loading anime data...",
		spinner:           sp,
	}
}

func (m Model) loadDataCmd() tea.Cmd {
	return func() tea.Msg {
		if m.scraper == nil {
			return dataLoadedMsg{err: fmt.Errorf("scraper not initialized")}
		}

		ctx := context.Background()
		data, err := m.scraper.GetHomepageData(ctx)
		if err != nil {
			return dataLoadedMsg{err: err}
		}

		return dataLoadedMsg{
			trending:   data.Popular,
			newRelease: data.NewReleases,
		}
	}
}

func (m Model) fetchAnimeDetailsCmd(slug string) tea.Cmd {
	return func() tea.Msg {
		if m.scraper == nil {
			return animeDetailsMsg{err: fmt.Errorf("scraper not initialized")}
		}

		ctx := context.Background()
		anime, err := m.scraper.GetAnime(ctx, slug)
		return animeDetailsMsg{anime: anime, err: err}
	}
}

func (m Model) fetchDetailCmd(slug string) tea.Cmd {
	return func() tea.Msg {
		if m.scraper == nil {
			return animeDetailUpdateMsg{slug: slug, err: fmt.Errorf("scraper not initialized")}
		}

		ctx := context.Background()
		anime, err := m.scraper.GetAnime(ctx, slug)
		return animeDetailUpdateMsg{slug: slug, anime: anime, err: err}
	}
}

func (m Model) fetchAllDetailsCmd() tea.Cmd {
	seen := make(map[string]bool)
	var cmds []tea.Cmd

	for _, anime := range m.trendingList {
		if anime.ID != "" && !seen[anime.ID] {
			seen[anime.ID] = true
			cmds = append(cmds, m.fetchDetailCmd(anime.ID))
		}
	}

	for _, anime := range m.newReleaseList {
		if anime.ID != "" && !seen[anime.ID] {
			seen[anime.ID] = true
			cmds = append(cmds, m.fetchDetailCmd(anime.ID))
		}
	}

	if len(cmds) == 0 {
		return nil
	}

	return tea.Batch(cmds...)
}

func (m Model) searchCmd(query string) tea.Cmd {
	return func() tea.Msg {
		if m.scraper == nil {
			return searchResultsMsg{err: fmt.Errorf("scraper not initialized")}
		}
		ctx := context.Background()
		results, err := m.scraper.Search(ctx, query)
		return searchResultsMsg{results: results, err: err}
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(titleTick(), m.loadDataCmd(), m.spinner.Tick)
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

func animeToCardData(anime Anime) CardData {
	subtitle := "Loading..."
	if anime.Year > 0 {
		subtitle = fmt.Sprintf("%d", anime.Year)
	}
	info := ""
	if anime.Seasons > 0 {
		info = CardInfoStyle.Render(fmt.Sprintf("%d Seasons", anime.Seasons))
	}
	return CardData{Name: anime.Name, Subtitle: subtitle, Info: info}
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
			return animeToCardData(m.trendingList[index])
		}
	case NewReleasesSection:
		if index < len(m.newReleaseList) {
			return animeToCardData(m.newReleaseList[index])
		}
	case SearchResultsSection:
		if index < len(m.searchResults) {
			return animeToCardData(m.searchResults[index])
		}
	}
	return CardData{}
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
