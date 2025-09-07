package views

import (
	"context"
	"github.com/hayasedb/hayase-cli/internal/models"
	"github.com/hayasedb/hayase-cli/internal/providers"
	"github.com/hayasedb/hayase-cli/internal/storage"
	"github.com/hayasedb/hayase-cli/internal/tui/navigation"
	"github.com/hayasedb/hayase-cli/internal/tui/ui"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type animeItem struct {
	result *models.SearchResult
}

func (i animeItem) Title() string { return i.result.Anime.Title }
func (i animeItem) Description() string {
	if i.result.Anime.Description != "" {
		return i.result.Anime.Description
	}
	return "No description available"
}
func (i animeItem) FilterValue() string { return i.result.Anime.Title }

type searchResultsMsg struct {
	results []*models.SearchResult
}

type errorMsg struct {
	err error
}

type AnimeView struct {
	list        list.Model
	searchInput ui.SearchInput
	delegate    *ui.CustomDelegate
	provider    providers.Provider
	config      *storage.Config
	state       *navigation.State
	searching   bool
	width       int
	height      int
	footer      *ui.Footer
}

func NewAnimeView(state *navigation.State, provider providers.Provider, config *storage.Config) *AnimeView {
	delegate := ui.NewCustomDelegate()
	delegate.SetShowSelection(false)
	l := list.New([]list.Item{}, delegate, 80, 24)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)

	searchInput := ui.NewSearchInput()

	return &AnimeView{
		list:        l,
		searchInput: searchInput,
		delegate:    delegate,
		provider:    provider,
		config:      config,
		state:       state,
		footer:      ui.NewFooter(),
	}
}

func (v *AnimeView) Update(msg tea.Msg) (*AnimeView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.searchInput.SetWidth(msg.Width)
		v.list.SetWidth(msg.Width)

	case tea.KeyMsg:
		oldValue := v.searchInput.Value()
		result, cmd := v.handleKeys(msg)
		if result.searchInput.IsFocused() && result.config.GetInstantSearch() && result.searchInput.Value() != oldValue {
			query := strings.TrimSpace(result.searchInput.Value())
			if query != "" && len(query) > 2 && !result.searching {
				result.searching = true
				spinnerCmd := result.searchInput.SetLoading(true)
				cmd = tea.Batch(cmd, result.search(), spinnerCmd)
			}
		}
		return result, cmd

	case searchResultsMsg:
		v.searching = false
		spinnerCmd := v.searchInput.SetLoading(false)
		items := v.createItems(msg.results)
		v.list.SetItems(items)

		return v, spinnerCmd

	case errorMsg:
		v.searching = false
		spinnerCmd := v.searchInput.SetLoading(false)
		return v, spinnerCmd
	}

	oldFocused := v.searchInput.IsFocused()
	oldValue := v.searchInput.Value()
	var cmd tea.Cmd
	searchInputPtr, cmd := v.searchInput.Update(msg)
	v.searchInput = *searchInputPtr

	if !oldFocused && v.searchInput.IsFocused() {
		v.delegate.SetShowSelection(false)
		return v, cmd
	}

	if v.searchInput.IsFocused() {
		if v.searchInput.Value() != oldValue && v.config.GetInstantSearch() {
			query := strings.TrimSpace(v.searchInput.Value())
			if query != "" && len(query) > 2 && !v.searching {
				v.searching = true
				spinnerCmd := v.searchInput.SetLoading(true)
				cmd = tea.Batch(cmd, v.search(), spinnerCmd)
			}
		}
	} else {
		var listCmd tea.Cmd
		v.list, listCmd = v.list.Update(msg)
		cmd = tea.Batch(cmd, listCmd)
	}
	return v, cmd
}

func (v *AnimeView) handleKeys(msg tea.KeyMsg) (*AnimeView, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		v.state.SetQuitting(true)
		return v, tea.Quit
	case "esc":
		if v.searchInput.IsFocused() {
			v.searchInput.SetValue("")
			v.list.SetItems([]list.Item{})
			return v, nil
		}
		return v, v.focusSearchInput()
	case "enter":
		if v.searchInput.IsFocused() {
			if query := strings.TrimSpace(v.searchInput.Value()); query != "" && !v.searching {
				v.searching = true
				v.focusListResults()
				spinnerCmd := v.searchInput.SetLoading(true)
				return v, tea.Batch(v.search(), spinnerCmd)
			}
		} else {
			return v.handleSelection()
		}
	case "down":
		if v.searchInput.IsFocused() && len(v.list.Items()) > 0 {
			return v, v.focusListResults()
		}
	case "up":
		if !v.searchInput.IsFocused() && v.list.Index() == 0 {
			return v, v.focusSearchInput()
		}
	}

	var cmd tea.Cmd
	if v.searchInput.IsFocused() {
		searchInputPtr, cmd := v.searchInput.Update(msg)
		v.searchInput = *searchInputPtr
		return v, cmd
	} else {
		v.list, cmd = v.list.Update(msg)
		return v, cmd
	}
}

func (v *AnimeView) handleSelection() (*AnimeView, tea.Cmd) {
	if item, ok := v.list.SelectedItem().(animeItem); ok {
		v.state.SetAnime(item.result.Anime)
		v.state.NavigateForward()
	}
	return v, nil
}

func (v *AnimeView) search() tea.Cmd {
	query := strings.TrimSpace(v.searchInput.Value())
	return func() tea.Msg {
		results, err := v.provider.Search(context.Background(), query)
		if err != nil {
			return errorMsg{err: err}
		}
		return searchResultsMsg{results: results}
	}
}

func (v *AnimeView) createItems(results []*models.SearchResult) []list.Item {
	items := make([]list.Item, len(results))
	for i, result := range results {
		items[i] = animeItem{result: result}
	}

	return items
}

func (v *AnimeView) focusSearchInput() tea.Cmd {
	v.searchInput.Focus()
	v.delegate.SetShowSelection(false)
	return textinput.Blink
}

func (v *AnimeView) focusListResults() tea.Cmd {
	v.searchInput.Blur()
	v.delegate.SetShowSelection(true)
	if len(v.list.Items()) > 0 {
		v.list.Select(0)
	}
	return nil
}

func (v *AnimeView) View() string {
	header := v.searchInput.View()

	if v.searchInput.IsFocused() {
		v.footer.SetKeys(ui.SearchModeKeys())
	} else {
		v.footer.SetKeys(ui.ListNavigationKeys())
	}
	footerView := v.footer.View()

	var content string
	switch {
	case len(v.list.Items()) > 0:
		v.list.SetHeight(v.height - lipgloss.Height(header) - lipgloss.Height(footerView))
		content = v.list.View()
	case v.searchInput.Value() != "":
		content = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).MarginLeft(2).Render("No results found")
	default:
		content = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).MarginLeft(2).Render("Type to search anime")
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, content, footerView)
}

func (v *AnimeView) Reset() {
	v.searchInput.Focus()
	v.searchInput.SetValue("")
	v.list.SetItems([]list.Item{})
	v.delegate.SetShowSelection(false)
	v.searching = false
}
