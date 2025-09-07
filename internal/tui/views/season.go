package views

import (
	"context"
	"fmt"
	"sort"

	"github.com/hayasedb/hayase-cli/internal/models"
	"github.com/hayasedb/hayase-cli/internal/providers"
	"github.com/hayasedb/hayase-cli/internal/tui/navigation"
	"github.com/hayasedb/hayase-cli/internal/tui/ui"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type seasonItem struct {
	seasonNum int
	episodes  []models.Episode
}

func (i seasonItem) Title() string {
	if i.seasonNum == 0 {
		return "Movies"
	}
	return fmt.Sprintf("Season %d", i.seasonNum)
}

func (i seasonItem) Description() string {
	count := len(i.episodes)
	if count == 1 {
		return "1 episode"
	}
	return fmt.Sprintf("%d episodes", count)
}

func (i seasonItem) FilterValue() string { return i.Title() }

type episodesLoadedMsg struct {
	err error
}

type SeasonView struct {
	list     list.Model
	state    *navigation.State
	provider providers.Provider
	loading  bool
	seasons  []int
	width    int
	height   int
	footer   *ui.Footer
}

func NewSeasonView(state *navigation.State, provider providers.Provider) *SeasonView {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 80, 24)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)

	return &SeasonView{
		list:     l,
		state:    state,
		provider: provider,
		footer:   ui.NewFooter(),
	}
}

func (v *SeasonView) Update(msg tea.Msg) (*SeasonView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.list.SetWidth(msg.Width)
		return v, nil

	case tea.KeyMsg:
		return v.handleKeys(msg)

	case episodesLoadedMsg:
		v.loading = false
		if msg.err == nil {
			v.populateSeasons()
		}
		return v, nil
	}

	if v.loading {
		return v, nil
	}

	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	return v, cmd
}

func (v *SeasonView) handleKeys(msg tea.KeyMsg) (*SeasonView, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		v.state.SetQuitting(true)
		return v, tea.Quit
	}

	if v.loading {
		return v, nil
	}

	switch msg.String() {
	case "esc":
		if v.list.FilterState() == list.Filtering {
			v.list.ResetFilter()
			return v, nil
		}
		v.state.NavigateBack()
		return v, nil
	case "enter":
		return v.handleSelection()
	}

	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	return v, cmd
}

func (v *SeasonView) handleSelection() (*SeasonView, tea.Cmd) {
	if v.list.Index() < len(v.seasons) {
		v.state.SetSeason(v.seasons[v.list.Index()])
		v.state.NavigateForward()
	}
	return v, nil
}

func (v *SeasonView) LoadAnime(anime *models.Anime) tea.Cmd {
	if len(v.seasons) == 0 || anime == nil || len(anime.Episodes) == 0 {
		v.loading = true
		return v.fetchEpisodes(anime)
	}
	v.populateSeasons()
	return nil
}

func (v *SeasonView) fetchEpisodes(anime *models.Anime) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := v.provider.GetEpisodes(ctx, anime)
		return episodesLoadedMsg{err: err}
	}
}

func (v *SeasonView) populateSeasons() {
	anime := v.state.GetAnime()
	if anime == nil {
		return
	}

	seasons := anime.GetAvailableSeasons()
	sort.Ints(seasons)
	v.seasons = seasons

	items := v.createSeasonItems(anime, seasons)
	v.list.SetItems(items)
	v.list.Title = anime.Title
}

func (v *SeasonView) createSeasonItems(anime *models.Anime, seasons []int) []list.Item {
	items := make([]list.Item, len(seasons))
	for i, seasonNum := range seasons {
		episodes := anime.GetEpisodesForSeason(seasonNum)
		items[i] = seasonItem{
			seasonNum: seasonNum,
			episodes:  episodes,
		}
	}
	return items
}

func (v *SeasonView) View() string {
	v.footer.SetKeys(ui.SeasonNavigationKeys())
	footerView := v.footer.View()

	var content string
	switch {
	case v.loading:
		content = v.renderLoading()
	case len(v.list.Items()) > 0:
		v.list.SetHeight(v.height - lipgloss.Height(footerView) - 1)
		content = lipgloss.NewStyle().MarginTop(1).Render(v.list.View())
	default:
		content = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginLeft(2).
			MarginTop(1).
			Render("No seasons available")
	}

	return lipgloss.JoinVertical(lipgloss.Left, content, footerView)
}

func (v *SeasonView) renderLoading() string {
	loading := lipgloss.JoinVertical(lipgloss.Center,
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			MarginLeft(2).
			Render("Loading seasons..."),
		"",
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginLeft(2).
			Render("Please wait..."))

	return lipgloss.NewStyle().
		Width(v.width).
		Height(v.height).
		AlignVertical(lipgloss.Top).
		Render(loading)
}
