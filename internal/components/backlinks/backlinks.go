package backlinks

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/takahashinaoki/obsidiantui/internal/vault"
)

type Model struct {
	backlinks []string
	cursor    int
	vault     *vault.Vault
	filePath  string
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
	Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "up")),
	Down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "down")),
	Enter:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
	Cancel: key.NewBinding(key.WithKeys("esc", "ctrl+b"), key.WithHelp("esc", "close")),
}

type FileSelectedMsg struct {
	Path string
}

type BacklinksClosedMsg struct{}

func New(v *vault.Vault) Model {
	return Model{
		vault: v,
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
			return m, func() tea.Msg {
				return BacklinksClosedMsg{}
			}

		case key.Matches(msg, DefaultKeyMap.Up):
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case key.Matches(msg, DefaultKeyMap.Down):
			if m.cursor < len(m.backlinks)-1 {
				m.cursor++
			}
			return m, nil

		case key.Matches(msg, DefaultKeyMap.Enter):
			if len(m.backlinks) > 0 && m.cursor < len(m.backlinks) {
				selected := m.backlinks[m.cursor]
				m.active = false
				return m, func() tea.Msg {
					return FileSelectedMsg{Path: selected}
				}
			}
			return m, nil
		}

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			clickedIndex := msg.Y - 3
			if clickedIndex >= 0 && clickedIndex < len(m.backlinks) {
				m.cursor = clickedIndex
				selected := m.backlinks[m.cursor]
				m.active = false
				return m, func() tea.Msg {
					return FileSelectedMsg{Path: selected}
				}
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	if !m.active {
		return ""
	}

	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("135")).
		Padding(0, 1)

	b.WriteString(titleStyle.Render("Backlinks") + "\n")

	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true)
	b.WriteString(subtitleStyle.Render("Files linking to: "+m.filePath) + "\n\n")

	if len(m.backlinks) > 0 {
		maxItems := m.height - 6
		if maxItems > len(m.backlinks) {
			maxItems = len(m.backlinks)
		}

		for i := 0; i < maxItems; i++ {
			link := m.backlinks[i]
			style := lipgloss.NewStyle()

			if i == m.cursor {
				style = style.Background(lipgloss.Color("62")).
					Foreground(lipgloss.Color("230"))
			} else {
				style = style.Foreground(lipgloss.Color("39"))
			}

			displayLink := link
			if len(displayLink) > m.width-6 {
				displayLink = "..." + displayLink[len(displayLink)-(m.width-9):]
			}

			b.WriteString(style.Width(m.width - 4).Render("  " + displayLink) + "\n")
		}

		if len(m.backlinks) > maxItems {
			moreStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
			b.WriteString(moreStyle.Render("  ... and more"))
		}
	} else {
		noLinksStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		b.WriteString(noLinksStyle.Render("  No backlinks found"))
	}

	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("135")).
		Padding(1).
		Width(m.width)

	return containerStyle.Render(b.String())
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *Model) Show(filePath string) {
	m.active = true
	m.filePath = filePath
	m.backlinks = m.vault.GetBacklinks(filePath)
	m.cursor = 0
}

func (m *Model) Hide() {
	m.active = false
}

func (m Model) Active() bool {
	return m.active
}

func (m Model) Backlinks() []string {
	return m.backlinks
}
