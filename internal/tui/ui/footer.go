package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type FooterKey struct {
	Key         string
	Description string
}

type Footer struct {
	keys []FooterKey
}

func NewFooter() *Footer {
	return &Footer{}
}

func (f *Footer) SetKeys(keys []FooterKey) {
	f.keys = keys
}

func (f *Footer) View() string {
	if len(f.keys) == 0 {
		return ""
	}

	parts := make([]string, len(f.keys))
	for i, key := range f.keys {
		parts[i] = key.Key + ": " + key.Description
	}

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginLeft(2).
		MarginBottom(1).
		MarginTop(1).
		Render(strings.Join(parts, " • "))
}

func SearchModeKeys() []FooterKey {
	return []FooterKey{
		{"enter", "search"},
		{"↓", "results"},
		{"esc", "clear"},
		{"q", "quit"},
	}
}

func ListNavigationKeys() []FooterKey {
	return []FooterKey{
		{"enter", "select"},
		{"↑↓", "navigate"},
		{"esc", "back"},
		{"q", "quit"},
	}
}

func SeasonNavigationKeys() []FooterKey {
	return []FooterKey{
		{"enter", "select"},
		{"/", "filter"},
		{"esc", "back"},
		{"q", "quit"},
	}
}

func EpisodeNavigationKeys() []FooterKey {
	return []FooterKey{
		{"enter", "select"},
		{"/", "filter"},
		{"esc", "back"},
		{"q", "quit"},
	}
}
