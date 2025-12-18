package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
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

func (m *Model) resetDetailsState() {
	m.activeSeason = 0
	m.seasonTabOffset = 0
	m.episodeCursor = 0
	m.episodeOffset = 0
}

func updateAnimeInSlice(list []Anime, slug string, updated *Anime) {
	for i := range list {
		if list[i].ID == slug {
			list[i] = *updated
			return
		}
	}
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

	case dataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.loadError = msg.err
			m.loadingStatus = "Failed to load data: " + msg.err.Error()
			return m, nil
		}

		m.trendingList = msg.trending
		m.newReleaseList = msg.newRelease

		p := m.sectionPaginators[TrendingSection]
		p.SetTotalPages(len(m.trendingList))
		m.sectionPaginators[TrendingSection] = p

		p = m.sectionPaginators[NewReleasesSection]
		p.SetTotalPages(len(m.newReleaseList))
		m.sectionPaginators[NewReleasesSection] = p

		return m, m.fetchAllDetailsCmd()

	case searchResultsMsg:
		m.searchLoading = false
		if msg.err != nil {
			return m, nil
		}
		m.searchResults = msg.results
		m.sectionCursors[SearchResultsSection] = 0
		m.resetSearchPaginator()
		return m, nil

	case searchDebounceMsg:
		if !m.searchActive || m.searchInput.Value() != msg.query {
			return m, nil
		}
		m.searchLoading = true
		return m, m.searchCmd(msg.query)

	case animeDetailsMsg:
		m.loadingDetails = false
		m.loadingStatus = ""
		if msg.err != nil {
			m.loadError = msg.err
			return m, nil
		}
		m.loadError = nil
		m.selectedAnime = msg.anime
		m.resetDetailsState()
		m.view = DetailsView
		return m, nil

	case animeDetailUpdateMsg:
		if msg.err != nil || msg.anime == nil {
			return m, nil
		}

		updateAnimeInSlice(m.trendingList, msg.slug, msg.anime)
		updateAnimeInSlice(m.newReleaseList, msg.slug, msg.anime)

		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

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
			if len(anime.Episodes) == 0 && anime.ID != "" {
				m.loadingDetails = true
				m.loadingStatus = "Loading " + anime.Name + "..."
				return m, m.fetchAnimeDetailsCmd(anime.ID)
			}
			m.selectedAnime = anime
			m.resetDetailsState()
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
			var inputCmd tea.Cmd
			m.searchInput, inputCmd = m.searchInput.Update(msg)
			query := m.searchInput.Value()
			if query != "" {
				return m, tea.Batch(inputCmd, searchDebounce(query))
			}
			return m, inputCmd
		}
	}

	return m, nil
}

func (m *Model) activateSearch() {
	m.searchActive = true
	m.searchInput.Focus()
	m.searchResults = nil
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
		m.searchLoading = false
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
				anime := m.searchResults[cursor]
				if len(anime.Episodes) == 0 && anime.ID != "" {
					m.loadingDetails = true
					m.loadingStatus = "Loading " + anime.Name + "..."
					m.searchActive = false
					m.searchInput.Blur()
					m.searchInput.SetValue("")
					return m, m.fetchAnimeDetailsCmd(anime.ID)
				}
				m.selectedAnime = &m.searchResults[cursor]
				m.resetDetailsState()
				m.view = DetailsView
				m.searchActive = false
				m.searchInput.Blur()
				m.searchInput.SetValue("")
			}
		}
		return m, nil

	default:
		var inputCmd tea.Cmd
		m.searchInput, inputCmd = m.searchInput.Update(msg)

		query := m.searchInput.Value()
		if query == "" {
			m.searchActive = false
			m.searchInput.Blur()
			m.searchResults = nil
			m.searchLoading = false
			m.activeSection = SearchSection
			return m, inputCmd
		}

		return m, tea.Batch(inputCmd, searchDebounce(query))
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
