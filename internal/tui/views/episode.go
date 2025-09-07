package views

import (
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

type episodeItem struct {
	episode *models.Episode
}

func (i episodeItem) Title() string {
	if i.episode.Title != "" {
		return i.episode.Title
	}
	return fmt.Sprintf("Episode %d", i.episode.Episode)
}

func (i episodeItem) Description() string {
	return fmt.Sprintf("Episode %d", i.episode.Episode)
}

func (i episodeItem) FilterValue() string {
	if i.episode.Title != "" {
		return i.episode.Title
	}
	return i.Title()
}

type EpisodeView struct {
	list     list.Model
	state    *navigation.State
	provider providers.Provider
	width    int
	height   int
	footer   *ui.Footer
}

func NewEpisodeView(state *navigation.State, provider providers.Provider) *EpisodeView {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 80, 24)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)

	return &EpisodeView{
		list:     l,
		state:    state,
		provider: provider,
		footer:   ui.NewFooter(),
	}
}

func (v *EpisodeView) Update(msg tea.Msg) (*EpisodeView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.list.SetWidth(msg.Width)
		return v, nil

	case tea.KeyMsg:
		return v.handleKeys(msg)
	}

	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	return v, cmd
}

func (v *EpisodeView) handleKeys(msg tea.KeyMsg) (*EpisodeView, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		v.state.SetQuitting(true)
		return v, tea.Quit
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

func (v *EpisodeView) handleSelection() (*EpisodeView, tea.Cmd) {
	if item, ok := v.list.SelectedItem().(episodeItem); ok {
		v.state.SetEpisode(item.episode)
		v.state.NavigateForward()
	}
	return v, nil
}

func (v *EpisodeView) LoadEpisodes() {
	anime := v.state.GetAnime()
	seasonNum := v.state.GetSeason()

	if anime == nil {
		return
	}

	episodes := anime.GetEpisodesForSeason(seasonNum)
	items := v.createEpisodeItems(episodes)

	v.list.SetItems(items)
	v.setListTitle(anime.Title, seasonNum)
}

func (v *EpisodeView) createEpisodeItems(episodes []models.Episode) []list.Item {
	sort.Slice(episodes, func(i, j int) bool {
		return episodes[i].Episode < episodes[j].Episode
	})

	items := make([]list.Item, len(episodes))
	for i, episode := range episodes {
		ep := episode
		items[i] = episodeItem{episode: &ep}
	}
	return items
}

func (v *EpisodeView) setListTitle(animeTitle string, seasonNum int) {
	if seasonNum == 0 {
		v.list.Title = fmt.Sprintf("%s > Movies", animeTitle)
	} else {
		v.list.Title = fmt.Sprintf("%s > Season %d", animeTitle, seasonNum)
	}
}

func (v *EpisodeView) View() string {
	v.footer.SetKeys(ui.EpisodeNavigationKeys())
	footerView := v.footer.View()

	var content string
	switch {
	case len(v.list.Items()) > 0:
		v.list.SetHeight(v.height - lipgloss.Height(footerView) - 1)
		content = lipgloss.NewStyle().MarginTop(1).Render(v.list.View())
	default:
		content = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginLeft(2).
			MarginTop(1).
			Render("No episodes available")
	}

	return lipgloss.JoinVertical(lipgloss.Left, content, footerView)
}
