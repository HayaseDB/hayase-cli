package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	Primary   = lipgloss.Color("#7C3AED")
	Secondary = lipgloss.Color("#A78BFA")
	Warning   = lipgloss.Color("#F59E0B")
	Text      = lipgloss.Color("#F9FAFB")
	TextMuted = lipgloss.Color("#9CA3AF")
	TextDim   = lipgloss.Color("#6B7280")
)

const (
	CardWidth         = 18
	CardHeight        = 3
	VisibleSeasonTabs = 8
)

var (
	InactiveTabStyle = lipgloss.NewStyle().
				Foreground(TextMuted).
				Padding(0, 1).
				MarginRight(1)

	ActiveTabStyle = lipgloss.NewStyle().
			Foreground(Text).
			Background(Primary).
			Padding(0, 1).
			MarginRight(1)

	TabArrowStyle = lipgloss.NewStyle().
			Foreground(TextDim).
			Padding(0, 1)
)

var (
	AppStyle = lipgloss.NewStyle().
			Padding(1, 2)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(TextMuted)

	DetailTitleStyle = lipgloss.NewStyle().
				Foreground(Primary).
				Bold(true)

	DetailMetaStyle = lipgloss.NewStyle().
			Foreground(TextMuted).
			MarginBottom(1)

	DetailDescStyle = lipgloss.NewStyle().
			Foreground(Text).
			MarginBottom(1)

	SectionTitleStyle = lipgloss.NewStyle().
				Foreground(Secondary).
				Bold(true).
				MarginBottom(1)

	HelpStyle = lipgloss.NewStyle().
			Foreground(TextDim).
			MarginTop(1)

	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(TextMuted)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(TextDim)

	GenreBadgeStyle = lipgloss.NewStyle().
			Foreground(Text).
			Background(Primary).
			Padding(0, 1).
			MarginRight(1)

	CursorStyle = lipgloss.NewStyle().
			Foreground(Primary)

	SearchBarStyle = lipgloss.NewStyle().
			Foreground(TextMuted).
			Padding(0, 1).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.NoColor{})

	SearchBarActiveStyle = lipgloss.NewStyle().
				Foreground(Text).
				Padding(0, 1).
				Border(lipgloss.NormalBorder(), false, false, true, false).
				BorderForeground(Primary)

	TextMutedStyle = lipgloss.NewStyle().
			Foreground(TextMuted)

	RowTitleStyle = lipgloss.NewStyle().
			Foreground(Secondary).
			Bold(true).
			MarginTop(1)

	CardStyle         = newCardStyle(TextDim)
	CardSelectedStyle = newCardStyle(Primary)

	CardTitleStyle = lipgloss.NewStyle().
			Foreground(Text).
			Bold(true)

	CardTitleSelectedStyle = lipgloss.NewStyle().
				Foreground(Primary).
				Bold(true)

	CardSubtitleStyle = lipgloss.NewStyle().
				Foreground(TextMuted)

	CardInfoStyle = lipgloss.NewStyle().
			Foreground(TextDim)

	ProgressStyle = lipgloss.NewStyle().
			Foreground(Primary)

	ProgressBgStyle = lipgloss.NewStyle().
			Foreground(TextDim)

	EpisodeRowStyle = lipgloss.NewStyle().
			Foreground(Text)

	EpisodeRowSelectedStyle = lipgloss.NewStyle().
				Foreground(Primary).
				Bold(true)

	EpisodeNumberStyle = lipgloss.NewStyle().
				Foreground(TextMuted)

	EpisodeDurationSubtleStyle = lipgloss.NewStyle().
					Foreground(TextDim)
)

func newCardStyle(borderColor lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(CardWidth).
		Height(CardHeight).
		Padding(0, 1)
}

func Dot() string {
	return lipgloss.NewStyle().Foreground(TextDim).Render(" • ")
}

func Truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func ScrollText(s string, width, offset int) string {
	if len(s) <= width {
		return s
	}
	padded := s + "   " + s
	maxOffset := len(s) + 3
	offset %= maxOffset
	end := offset + width
	if end > len(padded) {
		end = len(padded)
	}
	return padded[offset:end]
}

func ProgressBar(percent, width int) string {
	filled := (width * percent) / 100
	if filled > width {
		filled = width
	}
	empty := width - filled

	bar := ProgressStyle.Render(strings.Repeat("━", filled))
	bar += ProgressBgStyle.Render(strings.Repeat("─", empty))
	return bar
}
