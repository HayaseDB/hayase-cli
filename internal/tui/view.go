package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var content string

	switch m.view {
	case BrowseView:
		content = m.homeView()
	case DetailsView:
		content = m.detailsView()
	}

	return AppStyle.Render(content)
}

func (m Model) homeView() string {
	var b strings.Builder

	b.WriteString(m.renderSearchBar())
	b.WriteString("\n")

	if m.loading || m.loadingDetails {
		b.WriteString("\n")
		b.WriteString(m.spinner.View())
		b.WriteString(" ")
		b.WriteString(LoadingStyle.Render(m.loadingStatus))
		b.WriteString("\n")
		return b.String()
	}

	if m.loadError != nil {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(Warning).Render("Error: "))
		b.WriteString(LoadingStyle.Render(m.loadError.Error()))
		b.WriteString("\n")
	}

	if m.searchActive {
		b.WriteString(m.renderSearchResults())
	} else {
		b.WriteString(m.renderContinueWatching())
		b.WriteString("\n")

		if len(m.trendingList) > 0 {
			b.WriteString(m.renderRow("Trending Now", TrendingSection))
			b.WriteString("\n")
		}

		if len(m.newReleaseList) > 0 {
			b.WriteString(m.renderRow("New Releases", NewReleasesSection))
			b.WriteString("\n")
		}

		if len(m.trendingList) == 0 && len(m.newReleaseList) == 0 && m.loadError == nil {
			b.WriteString("\n")
			b.WriteString(TextMutedStyle.Render("No anime found. Try searching!"))
			b.WriteString("\n")
		}
	}

	if m.searchActive {
		b.WriteString(HelpStyle.Render(SearchHelp()))
	} else {
		b.WriteString(HelpStyle.Render(BrowseHelp()))
	}

	return b.String()
}

func (m Model) renderSearchBar() string {
	isSelected := m.activeSection == SearchSection || m.searchActive

	var style lipgloss.Style
	if isSelected {
		style = SearchBarActiveStyle
	} else {
		style = SearchBarStyle
	}

	return style.Render(m.searchInput.View())
}

func (m Model) renderSearchResults() string {
	var b strings.Builder

	b.WriteString(RowTitleStyle.Foreground(Primary).Render("Search Results"))
	b.WriteString("\n")

	if m.searchLoading {
		b.WriteString(m.spinner.View())
		b.WriteString(" ")
		b.WriteString(TextMutedStyle.Render("Searching..."))
		b.WriteString("\n")
		return b.String()
	}

	resultCount := len(m.searchResults)
	if resultCount == 0 {
		query := m.searchInput.Value()
		if query == "" {
			b.WriteString(TextMutedStyle.Render("Type to search..."))
		} else {
			b.WriteString(TextMutedStyle.Render("No anime found matching your search."))
		}
		b.WriteString("\n")
		return b.String()
	}

	cursor := m.sectionCursors[SearchResultsSection]
	p := m.sectionPaginators[SearchResultsSection]
	start, end := p.GetSliceBounds(resultCount)

	var cards []string
	for i := start; i < end; i++ {
		isSelected := i == cursor
		card := m.renderCard(SearchResultsSection, i, isSelected)
		cards = append(cards, card)
	}

	if len(cards) > 0 {
		row := lipgloss.JoinHorizontal(lipgloss.Top, cards...)
		b.WriteString(row)
	}

	if p.TotalPages > 1 {
		b.WriteString("\n")
		b.WriteString(TextMutedStyle.Render(fmt.Sprintf("Page %d/%d (%d results)", p.Page+1, p.TotalPages, resultCount)))
	}

	b.WriteString("\n")
	return b.String()
}

func (m Model) renderContinueWatching() string {
	isActiveRow := m.activeSection == ContinueSection
	titleStyle := RowTitleStyle
	if isActiveRow {
		titleStyle = titleStyle.Foreground(Primary)
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("Continue Watching"))
	b.WriteString("\n")

	if len(m.continueList) == 0 {
		b.WriteString(TextMutedStyle.Render("No watch history yet"))
		return b.String()
	}

	cursor := m.sectionCursors[ContinueSection]
	p := m.sectionPaginators[ContinueSection]
	listLen := len(m.continueList)
	start, end := p.GetSliceBounds(listLen)

	var cards []string
	for i := start; i < end; i++ {
		isSelected := isActiveRow && i == cursor
		card := m.renderCard(ContinueSection, i, isSelected)
		cards = append(cards, card)
	}

	if len(cards) > 0 {
		row := lipgloss.JoinHorizontal(lipgloss.Top, cards...)
		b.WriteString(row)
	}

	return b.String()
}

func (m Model) renderRow(title string, section Section) string {
	isActiveRow := m.activeSection == section
	cursor := m.sectionCursors[section]
	p := m.sectionPaginators[section]
	listLen := m.getSectionLength(section)

	var b strings.Builder

	titleStyle := RowTitleStyle
	if isActiveRow {
		titleStyle = titleStyle.Foreground(Primary)
	}
	if p.TotalPages > 1 {
		title = fmt.Sprintf("%s (%d/%d)", title, p.Page+1, p.TotalPages)
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")

	var cards []string
	start, end := p.GetSliceBounds(listLen)

	for i := start; i < end; i++ {
		isSelected := isActiveRow && i == cursor
		card := m.renderCard(section, i, isSelected)
		cards = append(cards, card)
	}

	if len(cards) > 0 {
		row := lipgloss.JoinHorizontal(lipgloss.Top, cards...)
		b.WriteString(row)
	}

	return b.String()
}

func (m Model) renderCard(section Section, index int, selected bool) string {
	data := m.getCardData(section, index)

	contentWidth := CardWidth - 2
	name := data.Name
	if selected && len(name) > contentWidth {
		name = ScrollText(name, contentWidth, m.titleScrollOffset)
	} else {
		name = Truncate(name, contentWidth)
	}

	var content strings.Builder

	titleStyle := CardTitleStyle
	if selected {
		titleStyle = CardTitleSelectedStyle
	}

	content.WriteString(titleStyle.Render(name))
	content.WriteString("\n")
	content.WriteString(CardSubtitleStyle.Render(data.Subtitle))
	content.WriteString("\n")
	content.WriteString(data.Info)

	cardStyle := CardStyle
	if selected {
		cardStyle = CardSelectedStyle
	}

	return cardStyle.Render(content.String()) + " "
}

func (m Model) detailsView() string {
	if m.selectedAnime == nil {
		return "No anime selected"
	}

	anime := m.selectedAnime
	var b strings.Builder

	b.WriteString(DetailTitleStyle.Render(anime.Name))
	b.WriteString("\n\n")

	meta := fmt.Sprintf("%d", anime.Year)
	meta += Dot() + fmt.Sprintf("%d Seasons", anime.Seasons)
	meta += Dot() + fmt.Sprintf("%d Episodes", len(anime.Episodes))
	b.WriteString(DetailMetaStyle.Render(meta))
	b.WriteString("\n\n")

	if len(anime.Genres) > 0 {
		for _, g := range anime.Genres {
			b.WriteString(GenreBadgeStyle.Render(g))
		}
		b.WriteString("\n\n")
	}

	if anime.Desc != "" {
		b.WriteString(DetailDescStyle.Render(anime.Desc))
		b.WriteString("\n\n")
	}

	if anime.Seasons >= 1 {
		b.WriteString(m.renderSeasonTabs())
		b.WriteString("\n\n")
	}

	b.WriteString(SectionTitleStyle.Render("Episodes"))
	b.WriteString("\n")

	episodes := m.getSeasonEpisodes()
	start := m.episodeOffset
	end := start + m.visibleEpisodes
	if end > len(episodes) {
		end = len(episodes)
	}

	for i := start; i < end; i++ {
		ep := episodes[i]
		epNum := fmt.Sprintf("%02d", ep.Number)

		if i == m.episodeCursor {
			b.WriteString(CursorStyle.Render("▸ "))
			b.WriteString(lipgloss.NewStyle().Foreground(Primary).Render(epNum))
			b.WriteString("  ")
			b.WriteString(EpisodeRowSelectedStyle.Render(ep.Title))
			b.WriteString("  ")
			b.WriteString(EpisodeDurationSubtleStyle.Render(ep.Duration))
		} else {
			b.WriteString("   ")
			b.WriteString(EpisodeNumberStyle.Render(epNum))
			b.WriteString("  ")
			b.WriteString(EpisodeRowStyle.Render(ep.Title))
			b.WriteString("  ")
			b.WriteString(EpisodeDurationSubtleStyle.Render(ep.Duration))
		}
		b.WriteString("\n")
	}

	if len(episodes) > m.visibleEpisodes {
		scrollInfo := fmt.Sprintf("%d-%d of %d", start+1, end, len(episodes))
		b.WriteString(SubtitleStyle.Render(scrollInfo))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render(DetailHelp()))

	return b.String()
}

func (m Model) renderSeasonTabs() string {
	if m.selectedAnime == nil || m.selectedAnime.Seasons < 1 {
		return ""
	}

	seasons := m.selectedAnime.Seasons

	start := m.seasonTabOffset
	end := start + VisibleSeasonTabs
	if end > seasons {
		end = seasons
	}

	var parts []string

	if start > 0 {
		parts = append(parts, TabArrowStyle.Render("◀"))
	}

	for i := start; i < end; i++ {
		label := fmt.Sprintf("Season %d", i+1)
		if i == m.activeSeason {
			parts = append(parts, ActiveTabStyle.Render(label))
		} else {
			parts = append(parts, InactiveTabStyle.Render(label))
		}
	}

	if end < seasons {
		parts = append(parts, TabArrowStyle.Render("▶"))
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, parts...)
}
