package search

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/takahashinaoki/obsidiantui/internal/vault"
)

var (
	searchTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("229")).
				Background(lipgloss.Color("57")).
				Padding(0, 1)
	searchInputStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62")).
				Padding(0, 1)
	searchSelectedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("62")).
				Foreground(lipgloss.Color("230"))
	searchNormalStyle   = lipgloss.NewStyle()
	searchMoreStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	searchNoResultStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	searchContainerBase = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62")).
				Padding(1)
)

type Model struct {
	textinput textinput.Model
	results   []string
	cursor    int
	vault     *vault.Vault
	width     int
	height    int
	active    bool
}

type KeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Enter  key.Binding
	Cancel key.Binding
}

var DefaultKeyMap = KeyMap{
	Up:     key.NewBinding(key.WithKeys("up", "ctrl+p"), key.WithHelp("up", "up")),
	Down:   key.NewBinding(key.WithKeys("down", "ctrl+n"), key.WithHelp("down", "down")),
	Enter:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	Cancel: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
}

type FileSelectedMsg struct {
	Path string
}

type SearchClosedMsg struct{}

func New(v *vault.Vault) Model {
	ti := textinput.New()
	ti.Placeholder = "Search files..."
	ti.CharLimit = 256
	ti.Width = 40

	return Model{
		textinput: ti,
		vault:     v,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DefaultKeyMap.Cancel):
			m.active = false
			m.textinput.Blur()
			return m, func() tea.Msg {
				return SearchClosedMsg{}
			}

		case key.Matches(msg, DefaultKeyMap.Up):
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case key.Matches(msg, DefaultKeyMap.Down):
			if m.cursor < len(m.results)-1 {
				m.cursor++
			}
			return m, nil

		case key.Matches(msg, DefaultKeyMap.Enter):
			if len(m.results) > 0 && m.cursor < len(m.results) {
				selected := m.results[m.cursor]
				m.active = false
				m.textinput.Blur()
				return m, func() tea.Msg {
					return FileSelectedMsg{Path: selected}
				}
			}
			return m, nil
		}

		var cmd tea.Cmd
		m.textinput, cmd = m.textinput.Update(msg)

		query := m.textinput.Value()
		if query != "" {
			m.results = m.vault.Search(query)
			if len(m.results) > 20 {
				m.results = m.results[:20]
			}
		} else {
			m.results = nil
		}
		m.cursor = 0

		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	if !m.active {
		return ""
	}

	var b strings.Builder

	b.WriteString(searchTitleStyle.Render("Search") + "\n")
	b.WriteString(searchInputStyle.Render(m.textinput.View()) + "\n")

	if len(m.results) > 0 {
		b.WriteString("\n")
		maxResults := m.height - 6
		if maxResults > len(m.results) {
			maxResults = len(m.results)
		}

		for i := 0; i < maxResults; i++ {
			result := m.results[i]
			var style lipgloss.Style

			if i == m.cursor {
				style = searchSelectedStyle
			} else {
				style = searchNormalStyle
			}

			if len(result) > m.width-4 {
				result = "..." + result[len(result)-(m.width-7):]
			}

			b.WriteString(style.Width(m.width - 2).Render("  " + result) + "\n")
		}

		if len(m.results) > maxResults {
			b.WriteString(searchMoreStyle.Render("  ... and more"))
		}
	} else if m.textinput.Value() != "" {
		b.WriteString(searchNoResultStyle.Render("  No results found"))
	}

	return searchContainerBase.Width(m.width).Render(b.String())
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.textinput.Width = width - 8
}

func (m *Model) Activate() tea.Cmd {
	m.active = true
	m.textinput.SetValue("")
	m.results = nil
	m.cursor = 0
	return m.textinput.Focus()
}

func (m *Model) Deactivate() {
	m.active = false
	m.textinput.Blur()
}

func (m Model) Active() bool {
	return m.active
}

func (m Model) Results() []string {
	return m.results
}
