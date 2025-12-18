package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func Run() error {
	p := tea.NewProgram(
		New(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	_, err := p.Run()
	return err
}
