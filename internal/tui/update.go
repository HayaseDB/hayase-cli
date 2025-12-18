package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) moveCursor(section Section, direction int) {
	cursor := m.sectionCursors[section]
	listLen := m.getSectionLength(section)
	newCursor := cursor + direction

	if newCursor < 0 || newCursor >= listLen {
		return
	}

	m.sectionCursors[section] = newCursor
	m.titleScrollOffset = 0

	p := m.sectionPaginators[section]
	start, end := p.GetSliceBounds(listLen)
	if newCursor < start {
		p.PrevPage()
	} else if newCursor >= end {
		p.NextPage()
	}
	m.sectionPaginators[section] = p
}

func (m *Model) resetSearchPaginator() {
	p := m.sectionPaginators[SearchResultsSection]
	p.Page = 0
	p.SetTotalPages(len(m.searchResults))
	m.sectionPaginators[SearchResultsSection] = p
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.visibleCards = m.calculateVisibleCards()

		for section, p := range m.sectionPaginators {
			p.PerPage = m.visibleCards
			p.SetTotalPages(m.getSectionLength(section))
			m.sectionPaginators[section] = p
		}

		m.visibleEpisodes = m.height - 14
		if m.visibleEpisodes < 3 {
			m.visibleEpisodes = 3
		}

		return m, nil

	case titleTickMsg:
		if m.view == BrowseView && m.activeSection != SearchSection {
			m.titleScrollOffset++
		}
		return m, titleTick()

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}

		if key.Matches(msg, m.keys.Quit) && !m.searchActive {
			m.quitting = true
			return m, tea.Quit
		}

		switch m.view {
		case BrowseView:
			if m.searchActive {
				return m.updateSearch(msg)
			}
			return m.updateBrowse(msg)
		case DetailsView:
			return m.updateDetails(msg)
		}
	}

	if m.searchActive {
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) updateBrowse(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.activeSection == SearchSection {
		return m.updateSearchSection(msg)
	}

	switch {
	case key.Matches(msg, m.keys.Filter):
		m.activateSearch()
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.activeSection > SearchSection {
			m.activeSection--
			m.titleScrollOffset = 0
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.activeSection < NewReleasesSection {
			m.activeSection++
			m.titleScrollOffset = 0
		}
		return m, nil

	case key.Matches(msg, m.keys.Left):
		m.moveCursor(m.activeSection, -1)
		return m, nil

	case key.Matches(msg, m.keys.Right):
		m.moveCursor(m.activeSection, 1)
		return m, nil

	case key.Matches(msg, m.keys.Enter):
		anime := m.getSelectedAnime()
		if anime != nil {
			m.selectedAnime = anime
			m.activeSeason = 0
			m.seasonTabOffset = 0
			m.episodeCursor = 0
			m.episodeOffset = 0
			m.view = DetailsView
		}
		return m, nil
	}

	return m, nil
}

func (m Model) updateSearchSection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Down):
		m.activeSection = ContinueSection
		m.titleScrollOffset = 0
		return m, nil

	case key.Matches(msg, m.keys.Enter):
		m.activateSearch()
		return m, nil

	default:
		keyStr := msg.String()
		if len(keyStr) == 1 && keyStr[0] >= 32 && keyStr[0] <= 126 {
			m.activateSearch()
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			m.searchResults = filterAnime(m.anime, m.searchInput.Value())
			m.resetSearchPaginator()
			return m, cmd
		}
	}

	return m, nil
}

func (m *Model) activateSearch() {
	m.searchActive = true
	m.searchInput.Focus()
	m.searchResults = m.anime
	m.activeSection = SearchResultsSection
	m.sectionCursors[SearchResultsSection] = 0
	m.titleScrollOffset = 0
	m.resetSearchPaginator()
}

func (m Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.searchActive = false
		m.searchInput.Blur()
		m.searchInput.SetValue("")
		m.searchResults = nil
		m.activeSection = SearchSection
		return m, nil

	case key.Matches(msg, m.keys.Left):
		m.moveCursor(SearchResultsSection, -1)
		return m, nil

	case key.Matches(msg, m.keys.Right):
		m.moveCursor(SearchResultsSection, 1)
		return m, nil

	case key.Matches(msg, m.keys.Enter):
		if len(m.searchResults) > 0 {
			cursor := m.sectionCursors[SearchResultsSection]
			if cursor < len(m.searchResults) {
				m.selectedAnime = &m.searchResults[cursor]
				m.activeSeason = 0
				m.seasonTabOffset = 0
				m.episodeCursor = 0
				m.episodeOffset = 0
				m.view = DetailsView
				m.searchActive = false
				m.searchInput.Blur()
				m.searchInput.SetValue("")
			}
		}
		return m, nil

	default:
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)

		query := m.searchInput.Value()
		if query == "" {
			m.searchActive = false
			m.searchInput.Blur()
			m.searchResults = nil
			m.activeSection = SearchSection
			return m, cmd
		}

		m.searchResults = filterAnime(m.anime, query)

		p := m.sectionPaginators[SearchResultsSection]
		p.SetTotalPages(len(m.searchResults))

		if m.sectionCursors[SearchResultsSection] >= len(m.searchResults) {
			m.sectionCursors[SearchResultsSection] = 0
			p.Page = 0
		}
		m.sectionPaginators[SearchResultsSection] = p

		return m, cmd
	}
}

func (m Model) updateDetails(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.selectedAnime == nil {
		return m, nil
	}

	episodes := m.getSeasonEpisodes()
	episodeCount := len(episodes)

	switch {
	case key.Matches(msg, m.keys.Back):
		m.view = BrowseView
		m.selectedAnime = nil
		return m, nil

	case key.Matches(msg, m.keys.Left):
		if m.selectedAnime.Seasons > 1 && m.activeSeason > 0 {
			m.activeSeason--
			m.episodeCursor = 0
			m.episodeOffset = 0
			if m.activeSeason < m.seasonTabOffset {
				m.seasonTabOffset = m.activeSeason
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Right):
		if m.selectedAnime.Seasons > 1 && m.activeSeason < m.selectedAnime.Seasons-1 {
			m.activeSeason++
			m.episodeCursor = 0
			m.episodeOffset = 0
			if m.activeSeason >= m.seasonTabOffset+VisibleSeasonTabs {
				m.seasonTabOffset = m.activeSeason - VisibleSeasonTabs + 1
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.episodeCursor > 0 {
			m.episodeCursor--
			if m.episodeCursor < m.episodeOffset {
				m.episodeOffset = m.episodeCursor
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.episodeCursor < episodeCount-1 {
			m.episodeCursor++
			if m.episodeCursor >= m.episodeOffset+m.visibleEpisodes {
				m.episodeOffset = m.episodeCursor - m.visibleEpisodes + 1
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Enter):
		return m, nil
	}

	return m, nil
}
