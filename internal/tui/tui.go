package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/hayasedb/hayase/internal/scraper"
)

func Run(s *scraper.Scraper) error {
	p := tea.NewProgram(
		New(s),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	_, err := p.Run()
	return err
}
