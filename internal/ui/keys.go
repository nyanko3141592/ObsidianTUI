package ui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Quit        key.Binding
	Help        key.Binding
	FocusNext   key.Binding
	FocusPrev   key.Binding
	Search      key.Binding
	Backlinks   key.Binding
	Save        key.Binding
	NewFile     key.Binding
	Delete      key.Binding
	Refresh     key.Binding
	ToggleView  key.Binding
	FollowLink  key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c", "q"),
			key.WithHelp("ctrl+c/q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		FocusNext: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next pane"),
		),
		FocusPrev: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev pane"),
		),
		Search: key.NewBinding(
			key.WithKeys("/", "ctrl+p"),
			key.WithHelp("/", "search"),
		),
		Backlinks: key.NewBinding(
			key.WithKeys("ctrl+b"),
			key.WithHelp("ctrl+b", "backlinks"),
		),
		Save: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "save"),
		),
		NewFile: key.NewBinding(
			key.WithKeys("ctrl+n"),
			key.WithHelp("ctrl+n", "new file"),
		),
		Delete: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "delete"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "refresh"),
		),
		ToggleView: key.NewBinding(
			key.WithKeys("ctrl+e"),
			key.WithHelp("ctrl+e", "toggle edit/preview"),
		),
		FollowLink: key.NewBinding(
			key.WithKeys("ctrl+]", "gd"),
			key.WithHelp("ctrl+]", "follow link"),
		),
	}
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit, k.Search, k.Save}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.FocusNext, k.FocusPrev, k.ToggleView},
		{k.Search, k.Backlinks, k.FollowLink},
		{k.Save, k.NewFile, k.Delete, k.Refresh},
		{k.Help, k.Quit},
	}
}
