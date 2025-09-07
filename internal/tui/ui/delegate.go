package ui

import (
	"io"

	"github.com/charmbracelet/bubbles/list"
)

type CustomDelegate struct {
	list.DefaultDelegate
	showSelection bool
}

func NewCustomDelegate() *CustomDelegate {
	d := list.NewDefaultDelegate()
	d.SetHeight(2)
	d.ShowDescription = true

	return &CustomDelegate{
		DefaultDelegate: d,
		showSelection:   true,
	}
}

func (d *CustomDelegate) SetShowSelection(show bool) {
	d.showSelection = show
}

func (d *CustomDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	renderIndex := index
	if !d.showSelection {
		renderIndex = -1
	}
	d.DefaultDelegate.Render(w, m, renderIndex, listItem)
}
