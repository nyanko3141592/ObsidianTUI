package ui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Quit       key.Binding
	Help       key.Binding
	CmdPalette key.Binding
	FocusNext  key.Binding
	FocusPrev  key.Binding
	FocusTree  key.Binding
	FocusEdit  key.Binding
	Search     key.Binding
	Backlinks  key.Binding
	Graph      key.Binding
	Tags       key.Binding
	Outline    key.Binding
	DailyNote  key.Binding
	Save       key.Binding
	NewFile    key.Binding
	Delete     key.Binding
	Refresh    key.Binding
	ToggleView key.Binding
	ViewEdit   key.Binding
	ViewPrev   key.Binding
	ViewSplit  key.Binding
	FollowLink key.Binding
	GoBack     key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c", "ctrl+q"),
			key.WithHelp("C-c", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		CmdPalette: key.NewBinding(
			key.WithKeys("f1", "ctrl+shift+p"),
			key.WithHelp("F1", "commands"),
		),
		FocusNext: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("Tab", "next pane"),
		),
		FocusPrev: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("S-Tab", "prev pane"),
		),
		FocusTree: key.NewBinding(
			key.WithKeys("alt+1", "ctrl+1"),
			key.WithHelp("M-1", "tree"),
		),
		FocusEdit: key.NewBinding(
			key.WithKeys("alt+2", "ctrl+2"),
			key.WithHelp("M-2", "editor"),
		),
		Search: key.NewBinding(
			key.WithKeys("/", "ctrl+p", "ctrl+f"),
			key.WithHelp("/", "search"),
		),
		Backlinks: key.NewBinding(
			key.WithKeys("ctrl+b"),
			key.WithHelp("C-b", "backlinks"),
		),
		Graph: key.NewBinding(
			key.WithKeys("ctrl+g"),
			key.WithHelp("C-g", "graph"),
		),
		Tags: key.NewBinding(
			key.WithKeys("ctrl+t"),
			key.WithHelp("C-t", "tags"),
		),
		Outline: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("C-l", "outline"),
		),
		DailyNote: key.NewBinding(
			key.WithKeys("alt+d"),
			key.WithHelp("M-d", "daily"),
		),
		Save: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("C-s", "save"),
		),
		NewFile: key.NewBinding(
			key.WithKeys("ctrl+n"),
			key.WithHelp("C-n", "new"),
		),
		Delete: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("C-d", "delete"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("ctrl+r", "f5"),
			key.WithHelp("C-r", "refresh"),
		),
		ToggleView: key.NewBinding(
			key.WithKeys("ctrl+e"),
			key.WithHelp("C-e", "cycle view"),
		),
		ViewEdit: key.NewBinding(
			key.WithKeys("alt+e"),
			key.WithHelp("M-e", "edit mode"),
		),
		ViewPrev: key.NewBinding(
			key.WithKeys("alt+p"),
			key.WithHelp("M-p", "preview"),
		),
		ViewSplit: key.NewBinding(
			key.WithKeys("alt+s"),
			key.WithHelp("M-s", "split"),
		),
		FollowLink: key.NewBinding(
			key.WithKeys("ctrl+]", "gd"),
			key.WithHelp("gd/C-]", "go to link"),
		),
		GoBack: key.NewBinding(
			key.WithKeys("ctrl+o", "ctrl+["),
			key.WithHelp("C-o", "go back"),
		),
	}
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.CmdPalette, k.Search, k.Save, k.ToggleView, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.FocusNext, k.FocusPrev, k.FocusTree, k.FocusEdit},
		{k.ToggleView, k.ViewEdit, k.ViewPrev, k.ViewSplit},
		{k.Search, k.Backlinks, k.Graph, k.Tags},
		{k.Outline, k.DailyNote, k.FollowLink, k.GoBack},
	}
}
