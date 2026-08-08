package tui

import "github.com/charmbracelet/bubbles/key"

type listKeyMap struct {
	Select   key.Binding
	New      key.Binding
	Delete   key.Binding
	Prune    key.Binding
	Lock     key.Binding
	Open     key.Binding
	Rename   key.Binding
	CycleTab key.Binding
	Refresh  key.Binding
	Quit     key.Binding
}

var listKeys = listKeyMap{
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select & cd"),
	),
	New: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete"),
	),
	Prune: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "prune"),
	),
	Lock: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "lock/unlock"),
	),
	Open: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open"),
	),
	Rename: key.NewBinding(
		key.WithKeys("R"),
		key.WithHelp("R", "rename"),
	),
	CycleTab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch view"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
