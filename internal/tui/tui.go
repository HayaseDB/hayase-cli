package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/hayasedb/hayase/internal/history"
	"github.com/hayasedb/hayase/internal/scraper"
)

func Run(s scraper.Interface, h *history.Manager) error {
	p := tea.NewProgram(
		New(s, h),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	_, err := p.Run()
	return err
}
