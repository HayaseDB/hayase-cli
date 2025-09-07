package ui

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SearchInput struct {
	textInput textinput.Model
	spinner   spinner.Model
	width     int
	loading   bool
}

func NewSearchInput() SearchInput {
	ti := textinput.New()
	ti.Placeholder = "Search anime..."
	ti.Focus()
	ti.Width = 40
	ti.Prompt = "> "

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return SearchInput{
		textInput: ti,
		spinner:   s,
		width:     42,
		loading:   false,
	}
}

func (s *SearchInput) Focus() tea.Cmd {
	return s.textInput.Focus()
}

func (s *SearchInput) Blur() {
	s.textInput.Blur()
}

func (s *SearchInput) SetLoading(loading bool) tea.Cmd {
	s.loading = loading
	if loading {
		s.textInput.Prompt = s.spinner.View() + " "
		return s.spinner.Tick
	} else {
		s.textInput.Prompt = "> "
		return nil
	}
}

func (s *SearchInput) Update(msg tea.Msg) (*SearchInput, tea.Cmd) {
	var cmds []tea.Cmd

	if mouse, ok := msg.(tea.MouseMsg); ok && mouse.Button == tea.MouseButtonLeft && mouse.Action == tea.MouseActionPress {
		if mouse.Y == 1 && mouse.X < s.width {
			if !s.textInput.Focused() {
				cmd := s.textInput.Focus()
				s.setCursorAtClick(mouse.X)
				cmds = append(cmds, cmd)
			} else {
				s.setCursorAtClick(mouse.X)
			}
		}
	}

	if s.loading {
		var spinnerCmd tea.Cmd
		s.spinner, spinnerCmd = s.spinner.Update(msg)
		cmds = append(cmds, spinnerCmd)
		s.textInput.Prompt = s.spinner.View()
	}

	var textCmd tea.Cmd
	s.textInput, textCmd = s.textInput.Update(msg)
	cmds = append(cmds, textCmd)

	return s, tea.Batch(cmds...)
}

func (s *SearchInput) setCursorAtClick(clickX int) {
	promptLen := 2

	textPos := clickX - promptLen
	if textPos < 0 {
		textPos = 0
	}

	value := s.textInput.Value()
	if textPos > len(value) {
		textPos = len(value)
	}

	s.textInput.SetCursor(textPos)
}

func (s *SearchInput) View() string {
	return lipgloss.NewStyle().MarginTop(1).MarginBottom(1).Render(s.textInput.View())
}

func (s *SearchInput) Value() string {
	return s.textInput.Value()
}

func (s *SearchInput) SetValue(value string) {
	s.textInput.SetValue(value)
}

func (s *SearchInput) SetWidth(width int) {
	s.textInput.Width = width - 6
	s.width = width - 4
}

func (s *SearchInput) IsFocused() bool {
	return s.textInput.Focused()
}
