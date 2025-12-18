package tui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Left   key.Binding
	Right  key.Binding
	Enter  key.Binding
	Back   key.Binding
	Quit   key.Binding
	Filter key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("←", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("→", "right"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
	}
}

func BrowseHelp() string {
	return HelpKeyStyle.Render("↑/↓") + HelpDescStyle.Render(" sections") +
		Dot() +
		HelpKeyStyle.Render("←/→") + HelpDescStyle.Render(" scroll") +
		Dot() +
		HelpKeyStyle.Render("/") + HelpDescStyle.Render(" search") +
		Dot() +
		HelpKeyStyle.Render("enter") + HelpDescStyle.Render(" select") +
		Dot() +
		HelpKeyStyle.Render("q") + HelpDescStyle.Render(" quit")
}

func SearchHelp() string {
	return HelpKeyStyle.Render("←/→") + HelpDescStyle.Render(" navigate") +
		Dot() +
		HelpKeyStyle.Render("enter") + HelpDescStyle.Render(" select") +
		Dot() +
		HelpKeyStyle.Render("esc") + HelpDescStyle.Render(" cancel")
}

func DetailHelp() string {
	return HelpKeyStyle.Render("↑/↓") + HelpDescStyle.Render(" episodes") +
		Dot() +
		HelpKeyStyle.Render("←/→") + HelpDescStyle.Render(" seasons") +
		Dot() +
		HelpKeyStyle.Render("enter") + HelpDescStyle.Render(" play") +
		Dot() +
		HelpKeyStyle.Render("esc") + HelpDescStyle.Render(" back")
}
